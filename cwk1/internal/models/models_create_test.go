package models

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupEmptyDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "retaildb-test-*.sqlite")
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
			t.Fatalf("create schema: %v", err)
		}
	}

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(path)
	})
	return db
}

func TestCreateProductSucceeds(t *testing.T) {
	db := setupEmptyDB(t)
	req := CreateProductRequest{
		ProductID:    "CP1",
		ProductName:  strPtr("Create Product"),
		Brand:        strPtr("BrandC"),
		ListingPrice: floatPtr(10.5),
		SalePrice:    floatPtr(8.25),
		Discount:     floatPtr(0.21),
		Revenue:      floatPtr(100.0),
		Description:  strPtr("A created product"),
		Rating:       floatPtr(4.0),
		Reviews:      floatPtr(3),
		LastVisited:  strPtr("2022-01-01 00:00:00"),
	}
	if err := CreateProduct(db, req); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	p, err := GetProductByID(db, "CP1")
	if err != nil {
		t.Fatalf("GetProductByID error: %v", err)
	}
	if p == nil {
		t.Fatalf("product not found")
	}
	if p.ProductName != "Create Product" {
		t.Fatalf("name mismatch: %v", p.ProductName)
	}
	if p.Brand != "BrandC" {
		t.Fatalf("brand mismatch: %v", p.Brand)
	}
	if p.ListingPrice != 10.5 {
		t.Fatalf("listing mismatch: %v", p.ListingPrice)
	}
	if p.SalePrice != 8.25 {
		t.Fatalf("sale mismatch: %v", p.SalePrice)
	}
}

func TestCreateProductDuplicate(t *testing.T) {
	db := setupEmptyDB(t)
	_ = CreateProduct(db, CreateProductRequest{ProductID: "DUP1", ProductName: strPtr("x")})
	if err := CreateProduct(db, CreateProductRequest{ProductID: "DUP1", ProductName: strPtr("y")}); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

// helpers
func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
