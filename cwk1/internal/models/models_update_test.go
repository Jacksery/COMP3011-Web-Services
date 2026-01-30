package models

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestUpdateModifiedFieldsNotFound(t *testing.T) {
	db := setupEmptyDB(t)
	err := UpdateModifiedFields(db, "NONEX", map[string]interface{}{"modified_product_name": "X"})
	if err == nil {
		t.Fatalf("expected error for non-existent update")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateModifiedFieldsSuccess(t *testing.T) {
	db := setupEmptyDB(t)
	// create baseline product
	if err := CreateProduct(db, CreateProductRequest{ProductID: "UPD1", ProductName: strPtr("Base")}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := UpdateModifiedFields(db, "UPD1", map[string]interface{}{"modified_product_name": "NewName"}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	p, err := GetProductByID(db, "UPD1")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if p == nil {
		t.Fatalf("expected product")
	}
	if p.ModifiedProductName == nil || *p.ModifiedProductName != "NewName" {
		t.Fatalf("modified name mismatch: %v", p.ModifiedProductName)
	}
}
