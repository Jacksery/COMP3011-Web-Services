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
	"TRUNCATE", "ATTACH", "DETACH", "REINDEX",
	"VACUUM", "PRAGMA",
}

// getSchema reads the full SQLite schema from sqlite_master so the LLM knows
// which tables and columns exist, plus sample rows so the LLM understands
// the actual data shape (types, 'None' strings, column semantics, etc.).
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

	// Append 3 sample rows per table so the LLM can see real data values
	tableRows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err == nil {
		defer func() {
			if cerr := tableRows.Close(); cerr != nil {
				log.Printf("warning: close table rows: %v", cerr)
			}
		}()
		for tableRows.Next() {
			var tbl string
			if err := tableRows.Scan(&tbl); err != nil {
				continue
			}
			sample, err := getSampleRows(db, tbl, 3)
			if err != nil {
				continue
			}
			if sample != "" {
				parts = append(parts, fmt.Sprintf("\n-- Sample rows from %s:\n%s", tbl, sample))
			}
		}
	}

	return strings.Join(parts, "\n"), nil
}

// getSampleRows returns a formatted string of sample rows from a table,
// ensuring a mix of populated and sparse/None rows so the LLM can see both.
func getSampleRows(db *sql.DB, table string, n int) (string, error) {
	// Discover columns first (table name comes from sqlite_master, not user input)
	colQuery := fmt.Sprintf("SELECT * FROM \"%s\" LIMIT 1", table) //nolint:gosec // table from sqlite_master
	probe, err := db.Query(colQuery)
	if err != nil {
		return "", err
	}
	cols, err := probe.Columns()
	if closeErr := probe.Close(); closeErr != nil {
		log.Printf("warning: close probe rows: %v", closeErr)
	}
	if err != nil {
		return "", err
	}

	// Find a non-PK column likely to have 'None' values for filtering
	// (skip first column which is usually the PK / product_id)
	filterCol := ""
	if len(cols) > 1 {
		filterCol = cols[1]
	}

	var allRows [][]string

	// Fetch rows where the filter column has real data (not 'None', not empty)
	if filterCol != "" {
		populatedQuery := fmt.Sprintf( //nolint:gosec // table/col from sqlite_master
			"SELECT * FROM \"%s\" WHERE \"%s\" != 'None' AND \"%s\" != '' AND \"%s\" IS NOT NULL LIMIT %d",
			table, filterCol, filterCol, filterCol, n,
		)
		populated, err := queryRowValues(db, populatedQuery, len(cols))
		if err == nil {
			allRows = append(allRows, populated...)
		}

		// Fetch rows where the filter column IS 'None' or empty
		sparseQuery := fmt.Sprintf( //nolint:gosec // table/col from sqlite_master
			"SELECT * FROM \"%s\" WHERE \"%s\" = 'None' OR \"%s\" = '' OR \"%s\" IS NULL LIMIT %d",
			table, filterCol, filterCol, filterCol, n/2+1,
		)
		sparse, err := queryRowValues(db, sparseQuery, len(cols))
		if err == nil {
			allRows = append(allRows, sparse...)
		}
	}

	// Fallback: if we got nothing, just grab the first n rows
	if len(allRows) == 0 {
		fallback := fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d", table, n) //nolint:gosec // table from sqlite_master
		fb, err := queryRowValues(db, fallback, len(cols))
		if err != nil {
			return "", err
		}
		allRows = fb
	}

	// Cap total sample rows
	maxRows := n * 2
	if len(allRows) > maxRows {
		allRows = allRows[:maxRows]
	}

	var lines []string
	lines = append(lines, "-- columns: "+strings.Join(cols, " | "))
	for _, vals := range allRows {
		lines = append(lines, "-- "+strings.Join(vals, " | "))
	}
	return strings.Join(lines, "\n"), nil
}

// queryRowValues runs a query and returns each row as a slice of string values.
func queryRowValues(db *sql.DB, query string, numCols int) ([][]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("warning: close rows: %v", cerr)
		}
	}()

	var result [][]string
	for rows.Next() {
		values := make([]interface{}, numCols)
		ptrs := make([]interface{}, numCols)
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		var vals []string
		for _, v := range values {
			switch t := v.(type) {
			case nil:
				vals = append(vals, "NULL")
			case []byte:
				vals = append(vals, string(t))
			default:
				vals = append(vals, fmt.Sprintf("%v", t))
			}
		}
		result = append(result, vals)
	}
	return result, nil
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

	// REPLACE is a valid SQLite string function, but "REPLACE INTO" is a write.
	// Only block the write form.
	if strings.Contains(upper, "REPLACE INTO") {
		return fmt.Errorf("query contains forbidden keyword: REPLACE INTO")
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
	prompt := fmt.Sprintf(`You are a SQL expert. Given the following SQLite database schema
and sample rows for each table:

%s

IMPORTANT DATA QUIRKS — read carefully before writing any query:

1. Python export artefacts: Many columns contain the literal string 'None' instead
   of SQL NULL for missing values. Numeric columns are stored as TEXT and use 'None'
   for missing data.
   - Always exclude rows where relevant columns = 'None' or = ''.
   - Cast numeric text columns with CAST(column AS REAL) for comparisons/ordering.

2. Modified columns: Some tables have modified_* columns that hold admin overrides.
   Always prefer the override: COALESCE(modified_column, original_column).

3. Duplicate data: The tables (info, brands, finance, reviews, traffic) are related
   by product_id but a product_id can appear MORE THAN ONCE in each table.
   Additionally, different product_ids can share the same product_name.
   - ALWAYS include i.product_id in the SELECT output so results can be distinguished.
   - When listing distinct products, GROUP BY i.product_id to avoid duplicate rows.
   - When the user asks about unique/distinct items by name, GROUP BY product_name
     instead and use MAX or MIN to pick one representative value per group.
   - When aggregating (SUM, AVG, COUNT, MAX, MIN), GROUP BY appropriately in a
     subquery first if needed, to avoid inflated results from duplicate rows.

4. CRITICAL — Use the sample rows above to understand which columns contain the data
   you need. Column names can be misleading. For example, in the reviews table the
   actual rating and review count are in the columns named 'real_rating' and
   'real_reviews', NOT 'rating' or 'reviews'. Always check the sample data to pick
   the correct columns.

Write a single READ-ONLY SQLite SELECT query that answers the user's question.
Rules:
- Output ONLY the raw SQL query, nothing else.
- Do NOT use INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, or any statement that modifies data.
- Limit results to 100 rows maximum.
- Use table aliases for readability.
- Always include product_id in the output columns.
- Deduplicate with GROUP BY on product_name when the user is asking about distinct products.
- Refer to the sample rows to pick the correct column for each concept.

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
