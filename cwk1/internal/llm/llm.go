package llm

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/genai"
)

// forbiddenKeywords are SQL keywords that indicate a write/DDL operation.
var forbiddenKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
	"REPLACE", "TRUNCATE", "ATTACH", "DETACH", "REINDEX",
	"VACUUM", "PRAGMA",
}

// getSchema reads the full SQLite schema from sqlite_master so the LLM knows
// which tables and columns exist.
func getSchema(db *sql.DB) (string, error) {
	rows, err := db.Query(`SELECT sql FROM sqlite_master WHERE type IN ('table','view') AND sql IS NOT NULL ORDER BY name`)
	if err != nil {
		return "", fmt.Errorf("querying schema: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("warning: close schema rows: %v", cerr)
		}
	}()

	var parts []string
	for rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return "", fmt.Errorf("scanning schema row: %w", err)
		}
		parts = append(parts, ddl+";")
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterating schema rows: %w", err)
	}
	return strings.Join(parts, "\n"), nil
}

// validateReadOnly rejects any SQL that is not a plain SELECT statement.
func validateReadOnly(query string) error {
	normalized := strings.TrimSpace(query)
	// Strip a trailing semicolon for checking purposes
	normalized = strings.TrimSuffix(normalized, ";")
	normalized = strings.TrimSpace(normalized)
	upper := strings.ToUpper(normalized)

	// Must start with SELECT or WITH (common-table-expressions that resolve to a SELECT)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("query must be a SELECT statement")
	}

	// Reject any forbidden keywords that could mutate data or schema
	for _, kw := range forbiddenKeywords {
		// Look for the keyword as a standalone token (preceded/followed by whitespace, parens, or start/end)
		if containsKeyword(upper, kw) {
			return fmt.Errorf("query contains forbidden keyword: %s", kw)
		}
	}

	// Reject multiple statements (semicolons in the middle)
	if strings.Contains(normalized, ";") {
		return fmt.Errorf("query contains multiple statements")
	}

	return nil
}

// containsKeyword checks if the uppercase SQL contains the keyword as a standalone word.
func containsKeyword(upperSQL, keyword string) bool {
	idx := 0
	for {
		pos := strings.Index(upperSQL[idx:], keyword)
		if pos == -1 {
			return false
		}
		abs := idx + pos
		before := abs == 0 || !isIdentChar(upperSQL[abs-1])
		after := abs+len(keyword) >= len(upperSQL) || !isIdentChar(upperSQL[abs+len(keyword)])
		if before && after {
			return true
		}
		idx = abs + len(keyword)
	}
}

func isIdentChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// execReadOnlyQuery runs a validated SELECT and returns the result rows as a
// slice of column-name → value maps.
func execReadOnlyQuery(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("warning: close query rows: %v", cerr)
		}
	}()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			val := values[i]
			// Convert []byte to string for JSON friendliness
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return results, nil
}

// Query takes a natural-language question, asks Gemini to produce a read-only
// SQL query for the RetailDB SQLite database, validates it, executes it, and
// returns the generated SQL together with the result rows.
// model can be empty, in which case "gemini-2.0-flash-lite" is used.
func Query(ctx context.Context, db *sql.DB, apiKey, model, question string) (string, []map[string]interface{}, error) {
	if model == "" {
		model = "gemini-3-flash-preview"
	}
	// 1. Fetch live schema
	schema, err := getSchema(db)
	if err != nil {
		return "", nil, fmt.Errorf("reading schema: %w", err)
	}

	// 2. Build prompt
	prompt := fmt.Sprintf(`You are a SQL expert. Given the following SQLite database schema:

%s

IMPORTANT: This dataset was exported from Python. Many columns contain the literal
string 'None' instead of SQL NULL for missing values. Numeric columns (listing_price,
sale_price, discount, revenue, rating, reviews, etc.) store numbers as TEXT and use
the string 'None' for missing data.

When writing queries:
- Always exclude rows where relevant columns equal the string 'None' or are empty.
  For example: WHERE f.revenue != 'None' AND f.revenue != ''
- Cast text numeric columns with CAST(column AS REAL) when doing numeric comparisons or ordering.
- Use COALESCE with the modified_* column first, e.g. COALESCE(i.modified_product_name, i.product_name)
  to pick up any admin edits.

Write a single READ-ONLY SQLite SELECT query that answers the user's question.
Rules:
- Output ONLY the raw SQL query, nothing else.
- Do NOT use INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, or any statement that modifies data.
- Limit results to 100 rows maximum.
- Use table aliases for readability.

User question: %s`, schema, question)

	// 3. Call Gemini — set GEMINI_API_KEY env so the client picks it up
	if err := os.Setenv("GEMINI_API_KEY", apiKey); err != nil {
		return "", nil, fmt.Errorf("setting GEMINI_API_KEY: %w", err)
	}
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", nil, fmt.Errorf("creating genai client: %w", err)
	}

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		return "", nil, fmt.Errorf("gemini generate: %w", err)
	}

	// Extract text from response
	sqlQuery := ""
	if result != nil && len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		for _, part := range result.Candidates[0].Content.Parts {
			if part.Text != "" {
				sqlQuery += part.Text
			}
		}
	}

	sqlQuery = strings.TrimSpace(sqlQuery)
	// Strip markdown code fences the model sometimes wraps SQL in
	sqlQuery = strings.TrimPrefix(sqlQuery, "```sql")
	sqlQuery = strings.TrimPrefix(sqlQuery, "```")
	sqlQuery = strings.TrimSuffix(sqlQuery, "```")
	sqlQuery = strings.TrimSpace(sqlQuery)

	if sqlQuery == "" {
		return "", nil, fmt.Errorf("gemini returned an empty query")
	}

	// 4. Validate the query is read-only
	if err := validateReadOnly(sqlQuery); err != nil {
		return sqlQuery, nil, fmt.Errorf("unsafe query rejected: %w", err)
	}

	// 5. Execute the query
	rows, err := execReadOnlyQuery(db, sqlQuery)
	if err != nil {
		return sqlQuery, nil, fmt.Errorf("query execution: %w", err)
	}

	return sqlQuery, rows, nil
}
