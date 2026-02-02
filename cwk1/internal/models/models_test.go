package models

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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

func TestCreateAndGetProduct(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	req := CreateProductRequest{
		ProductID:    "p-1",
		ProductName:  strPtr("Widget"),
		Brand:        strPtr("Acme"),
		Description:  strPtr("A test widget"),
		ListingPrice: floatPtr(19.99),
		SalePrice:    floatPtr(9.99),
		Discount:     floatPtr(10.0),
		Revenue:      floatPtr(100.5),
		Rating:       floatPtr(4.5),
		Reviews:      floatPtr(12.0),
		LastVisited:  strPtr("2026-02-02"),
	}

	if err := CreateProduct(db, req); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	p, err := GetProductByID(db, "p-1")
	if err != nil {
		t.Fatalf("GetProductByID failed: %v", err)
	}
	if p == nil {
		t.Fatalf("expected product, got nil")
	}
	if p.ProductID != "p-1" {
		t.Fatalf("unexpected product_id: %s", p.ProductID)
	}
	if p.ProductName != "Widget" {
		t.Fatalf("unexpected product_name: %s", p.ProductName)
	}
	if p.Brand != "Acme" {
		t.Fatalf("unexpected brand: %s", p.Brand)
	}
}

func TestGetProductsPagination(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("p-%d", i)
		if err := CreateProduct(db, CreateProductRequest{ProductID: id}); err != nil {
			t.Fatalf("CreateProduct failed: %v", err)
		}
	}

	res, err := GetProducts(db, 2, 1)
	if err != nil {
		t.Fatalf("GetProducts failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 products, got %d", len(res))
	}
}

func TestUpdateAndDeleteProduct(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	if err := CreateProduct(db, CreateProductRequest{ProductID: "p-x"}); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	updates := map[string]interface{}{
		"modified_product_name":  "NewName",
		"modified_brand":         "NewBrand",
		"modified_listing_price": 55.5,
	}
	if err := UpdateModifiedFields(db, "p-x", updates); err != nil {
		t.Fatalf("UpdateModifiedFields failed: %v", err)
	}
	p, err := GetProductByID(db, "p-x")
	if err != nil {
		t.Fatalf("GetProductByID failed: %v", err)
	}
	if p.ModifiedProductName == nil || *p.ModifiedProductName != "NewName" {
		t.Fatalf("modified product name not set: %v", p.ModifiedProductName)
	}
	if p.ModifiedBrand == nil || *p.ModifiedBrand != "NewBrand" {
		t.Fatalf("modified brand not set: %v", p.ModifiedBrand)
	}
	if p.ModifiedListingPrice == nil || *p.ModifiedListingPrice != 55.5 {
		t.Fatalf("modified listing price not set: %v", p.ModifiedListingPrice)
	}

	if err := DeleteProduct(db, "p-x"); err != nil {
		t.Fatalf("DeleteProduct failed: %v", err)
	}
	// deleting again should return sql.ErrNoRows
	if err := DeleteProduct(db, "p-x"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows on deleting non-existent, got: %v", err)
	}
}

// helpers
func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
