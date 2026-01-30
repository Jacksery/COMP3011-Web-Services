package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestServer(t *testing.T) (baseURL string, cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", "retaildb-int-*.sqlite")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = f.Close()
	path := f.Name()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	schema := []string{
		`CREATE TABLE info (
		  product_name VARCHAR(64),
		  product_id VARCHAR(50),
		  description VARCHAR(512),
		  modified_product_name VARCHAR,
		  modified_description VARCHAR
		);`,
		`CREATE TABLE brands (
		  product_id VARCHAR(50),
		  brand VARCHAR(50),
		  modified_brand VARCHAR
		);`,
		`CREATE TABLE finance (
		  product_id VARCHAR(50),
		  listing_price REAL,
		  sale_price REAL,
		  discount REAL,
		  revenue REAL,
		  modified_listing_price REAL,
		  modified_sale_price REAL,
		  modified_discount REAL,
		  modified_revenue REAL
		);`,
		`CREATE TABLE reviews (
		  product_id VARCHAR(50),
		  rating VARCHAR(50),
		  reviews VARCHAR(50),
		  "Hour" REAL,
		  "minute" REAL,
		  real_rating REAL,
		  real_reviews REAL,
		  "Unnamed: 7" VARCHAR(50)
		);`,
		`CREATE TABLE traffic (
		  product_id VARCHAR(50),
		  last_visited TEXT(50),
		  modified_last_visited TEXT
		);`,
	}
	for _, s := range schema {
		if _, err := db.Exec(s); err != nil {
			db.Close()
			t.Fatalf("create schema: %v", err)
		}
	}

	// start server with this db
	r := gin.New()
	RegisterRoutes(r, db)
	ts := httptest.NewServer(r)

	cleanup = func() {
		ts.Close()
		_ = db.Close()
		_ = os.Remove(path)
	}
	return ts.URL, cleanup
}

func TestIntegrationCRUD(t *testing.T) {
	base, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{}

	// 1) login
	loginBody := map[string]string{"username": "admin", "password": "password"}
	bb, _ := json.Marshal(loginBody)
	resp, err := client.Post(base+"/auth/login", "application/json", bytes.NewBuffer(bb))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed: %d %s", resp.StatusCode, string(b))
	}
	var lr map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	tok, ok := lr["token"]
	if !ok || tok == "" {
		t.Fatalf("no token returned")
	}

	// helper to make auth requests
	reqAuth := func(method, path string, body interface{}) (*http.Response, []byte, error) {
		var rb io.Reader
		if body != nil {
			bts, _ := json.Marshal(body)
			rb = bytes.NewBuffer(bts)
		}
		req, _ := http.NewRequest(method, base+path, rb)
		req.Header.Set("Authorization", "Bearer "+tok)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return res, b, nil
	}

	// 2) create product
	create := map[string]interface{}{
		"product_id":    "INT1",
		"product_name":  "Integration Item",
		"brand":         "IntBrand",
		"listing_price": 42.5,
	}
	res, body, err := reqAuth("POST", "/admin/products", create)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create failed: %d %s", res.StatusCode, string(body))
	}

	// 3) get product
	resGet, err := http.Get(base + "/products/INT1")
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	bGet, _ := io.ReadAll(resGet.Body)
	resGet.Body.Close()
	if resGet.StatusCode != 200 {
		t.Fatalf("get failed: %d %s", resGet.StatusCode, string(bGet))
	}
	var got map[string]interface{}
	if err := json.Unmarshal(bGet, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got["product_id"] != "INT1" {
		t.Fatalf("unexpected id: %v", got["product_id"])
	}

	// 4) update modified name
	res, body, err = reqAuth("PUT", "/admin/products/INT1", map[string]interface{}{"modified_product_name": "NewInt"})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("update failed: %d %s", res.StatusCode, string(body))
	}

	// verify updated
	res2, err := http.Get(base + "/products/INT1")
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	b2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	var got2 map[string]interface{}
	_ = json.Unmarshal(b2, &got2)
	if got2["modified_product_name"] != "NewInt" {
		t.Fatalf("modified name not set: %v", got2["modified_product_name"])
	}

	// 5) delete
	res, body, err = reqAuth("DELETE", "/admin/products/INT1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("delete failed: %d %s", res.StatusCode, string(body))
	}

	// 6) get should 404
	res3, _ := http.Get(base + "/products/INT1")
	if res3.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", res3.StatusCode)
	}
}
