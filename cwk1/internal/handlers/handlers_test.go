package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"retaildb-service/internal/auth"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	ddls := []string{
		`CREATE TABLE info (product_id TEXT PRIMARY KEY, product_name TEXT, modified_product_name TEXT, description TEXT, modified_description TEXT);`,
		`CREATE TABLE brands (product_id TEXT PRIMARY KEY, brand TEXT, modified_brand TEXT);`,
		`CREATE TABLE finance (product_id TEXT PRIMARY KEY, listing_price TEXT, sale_price TEXT, discount TEXT, revenue TEXT, modified_listing_price TEXT, modified_sale_price TEXT, modified_discount TEXT, modified_revenue TEXT);`,
		`CREATE TABLE reviews (product_id TEXT PRIMARY KEY, rating TEXT, reviews TEXT, real_rating REAL, real_reviews REAL);`,
		`CREATE TABLE traffic (product_id TEXT PRIMARY KEY, last_visited TEXT, modified_last_visited TEXT);`,
	}
	for _, d := range ddls {
		if _, err := db.Exec(d); err != nil {
			t.Fatalf("failed to exec ddl: %v", err)
		}
	}
	return db
}

func TestHandlers_HelloAndAuthAndAdminFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, db)

	// healthz
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz returned %d", rec.Code)
	}

	// login
	loginBody := map[string]string{"username": "admin", "password": "password"}
	b, _ := json.Marshal(loginBody)
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login returned %d", rec.Code)
	}
	var lr map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &lr); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	tok, ok := lr["token"]
	if !ok || tok == "" {
		t.Fatalf("no token returned")
	}

	// creating product without auth should fail
	prod := map[string]interface{}{"product_id": "h-1", "product_name": "HTest"}
	pb, _ := json.Marshal(prod)
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/admin/products", bytes.NewReader(pb))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when creating product without auth, got %d", rec.Code)
	}

	// create product with auth
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/admin/products", bytes.NewReader(pb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating product with auth, got %d", rec.Code)
	}

	// get product
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/products/h-1", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when getting product, got %d", rec.Code)
	}

	// update modified fields
	updates := map[string]interface{}{"modified_product_name": "NewH"}
	ub, _ := json.Marshal(updates)
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/admin/products/h-1", bytes.NewReader(ub))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when updating product, got %d", rec.Code)
	}

	// delete product
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/admin/products/h-1", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when deleting product, got %d", rec.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, db)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/admin/products/nonexist", nil)
	req.Header.Set("Authorization", "Bearer bad.token.here")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad token, got %d", rec.Code)
	}
}

// a small helper to ensure the auth package initializes deterministically in tests
func init() {
	auth.Init()
}
