package models

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDeleteProduct(t *testing.T) {
	db := setupEmptyDB(t)
	// create a product
	req := CreateProductRequest{ProductID: "DEL1", ProductName: strPtr("ToDelete")}
	if err := CreateProduct(db, req); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// delete it
	if err := DeleteProduct(db, "DEL1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	// ensure it's gone
	p, err := GetProductByID(db, "DEL1")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil after delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	db := setupEmptyDB(t)
	if err := DeleteProduct(db, "NOPE"); err == nil {
		t.Fatalf("expected error for non-existent delete")
	} else {
		if err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	}
}
