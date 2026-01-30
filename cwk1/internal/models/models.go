package models

import (
	"database/sql"
	"fmt"
	"strconv"
)

type Product struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name,omitempty"`
	Brand        string  `json:"brand,omitempty"`
	ListingPrice float64 `json:"listing_price,omitempty"`
	SalePrice    float64 `json:"sale_price,omitempty"`
	Discount     float64 `json:"discount,omitempty"`
	Revenue      float64 `json:"revenue,omitempty"`
	Description  string  `json:"description,omitempty"`
	Rating       float64 `json:"rating,omitempty"`
	Reviews      float64 `json:"reviews,omitempty"`
	LastVisited  string  `json:"last_visited,omitempty"`

	ModifiedProductName  *string  `json:"modified_product_name,omitempty"`
	ModifiedDescription  *string  `json:"modified_description,omitempty"`
	ModifiedBrand        *string  `json:"modified_brand,omitempty"`
	ModifiedListingPrice *float64 `json:"modified_listing_price,omitempty"`
	ModifiedSalePrice    *float64 `json:"modified_sale_price,omitempty"`
	ModifiedDiscount     *float64 `json:"modified_discount,omitempty"`
	ModifiedRevenue      *float64 `json:"modified_revenue,omitempty"`
	ModifiedLastVisited  *string  `json:"modified_last_visited,omitempty"`
}

func GetProducts(db *sql.DB, limit, offset int) ([]Product, error) {
	rows, err := db.Query(`
		SELECT i.product_id, coalesce(i.modified_product_name, i.product_name) as product_name,
		       coalesce(b.modified_brand, b.brand) as brand,
		       coalesce(f.modified_listing_price, f.listing_price) as listing_price,
		       coalesce(f.modified_sale_price, f.sale_price) as sale_price,
		       coalesce(f.modified_discount, f.discount) as discount,
		       coalesce(f.modified_revenue, f.revenue) as revenue,
		       coalesce(i.modified_description, i.description) as description,
		       r.real_rating as rating, r.real_reviews as reviews,
		       coalesce(t.modified_last_visited, t.last_visited) as last_visited
		FROM info i
		LEFT JOIN brands b ON i.product_id = b.product_id
		LEFT JOIN finance f ON i.product_id = f.product_id
		LEFT JOIN reviews r ON i.product_id = r.product_id
		LEFT JOIN traffic t ON i.product_id = t.product_id
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Product
	for rows.Next() {
		var p Product
		var brand sql.NullString
		var listingStr, saleStr, discountStr, revenueStr sql.NullString
		var ratingStr, reviewsStr sql.NullString
		var description sql.NullString
		var lastVisited sql.NullString
		err := rows.Scan(&p.ProductID, &p.ProductName, &brand, &listingStr, &saleStr, &discountStr, &revenueStr, &description, &ratingStr, &reviewsStr, &lastVisited)
		if err != nil {
			return nil, err
		}
		if brand.Valid {
			p.Brand = brand.String
		}
		if listingStr.Valid && listingStr.String != "None" && listingStr.String != "" {
			if v, e := strconv.ParseFloat(listingStr.String, 64); e == nil {
				p.ListingPrice = v
			}
		}
		if saleStr.Valid && saleStr.String != "None" && saleStr.String != "" {
			if v, e := strconv.ParseFloat(saleStr.String, 64); e == nil {
				p.SalePrice = v
			}
		}
		if discountStr.Valid && discountStr.String != "None" && discountStr.String != "" {
			if v, e := strconv.ParseFloat(discountStr.String, 64); e == nil {
				p.Discount = v
			}
		}
		if revenueStr.Valid && revenueStr.String != "None" && revenueStr.String != "" {
			if v, e := strconv.ParseFloat(revenueStr.String, 64); e == nil {
				p.Revenue = v
			}
		}
		if description.Valid {
			p.Description = description.String
		}
		if ratingStr.Valid && ratingStr.String != "None" && ratingStr.String != "" {
			if v, e := strconv.ParseFloat(ratingStr.String, 64); e == nil {
				p.Rating = v
			}
		}
		if reviewsStr.Valid && reviewsStr.String != "None" && reviewsStr.String != "" {
			if v, e := strconv.ParseFloat(reviewsStr.String, 64); e == nil {
				p.Reviews = v
			}
		}
		if lastVisited.Valid {
			p.LastVisited = lastVisited.String
		}
		res = append(res, p)
	}
	return res, nil
}

func GetProductByID(db *sql.DB, id string) (*Product, error) {
	row := db.QueryRow(`
		SELECT i.product_id, coalesce(i.modified_product_name, i.product_name) as product_name,
		       coalesce(b.modified_brand, b.brand) as brand,
		       coalesce(f.modified_listing_price, f.listing_price) as listing_price,
		       coalesce(f.modified_sale_price, f.sale_price) as sale_price,
		       coalesce(f.modified_discount, f.discount) as discount,
		       coalesce(f.modified_revenue, f.revenue) as revenue,
		       coalesce(i.modified_description, i.description) as description,
		       r.real_rating as rating, r.real_reviews as reviews,
		       coalesce(t.modified_last_visited, t.last_visited) as last_visited
		FROM info i
		LEFT JOIN brands b ON i.product_id = b.product_id
		LEFT JOIN finance f ON i.product_id = f.product_id
		LEFT JOIN reviews r ON i.product_id = r.product_id
		LEFT JOIN traffic t ON i.product_id = t.product_id
		WHERE i.product_id = ?`, id)
	var p Product
	var brand sql.NullString
	var listingStr, saleStr, discountStr, revenueStr sql.NullString
	var ratingStr, reviewsStr sql.NullString
	var description sql.NullString
	var lastVisited sql.NullString
	err := row.Scan(&p.ProductID, &p.ProductName, &brand, &listingStr, &saleStr, &discountStr, &revenueStr, &description, &ratingStr, &reviewsStr, &lastVisited)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if brand.Valid {
		p.Brand = brand.String
	}
	if listingStr.Valid && listingStr.String != "None" && listingStr.String != "" {
		if v, e := strconv.ParseFloat(listingStr.String, 64); e == nil {
			p.ListingPrice = v
		}
	}
	if saleStr.Valid && saleStr.String != "None" && saleStr.String != "" {
		if v, e := strconv.ParseFloat(saleStr.String, 64); e == nil {
			p.SalePrice = v
		}
	}
	if discountStr.Valid && discountStr.String != "None" && discountStr.String != "" {
		if v, e := strconv.ParseFloat(discountStr.String, 64); e == nil {
			p.Discount = v
		}
	}
	if revenueStr.Valid && revenueStr.String != "None" && revenueStr.String != "" {
		if v, e := strconv.ParseFloat(revenueStr.String, 64); e == nil {
			p.Revenue = v
		}
	}
	if description.Valid {
		p.Description = description.String
	}
	if ratingStr.Valid && ratingStr.String != "None" && ratingStr.String != "" {
		if v, e := strconv.ParseFloat(ratingStr.String, 64); e == nil {
			p.Rating = v
		}
	}
	if reviewsStr.Valid && reviewsStr.String != "None" && reviewsStr.String != "" {
		if v, e := strconv.ParseFloat(reviewsStr.String, 64); e == nil {
			p.Reviews = v
		}
	}
	if lastVisited.Valid {
		p.LastVisited = lastVisited.String
	}
	row2 := db.QueryRow(`SELECT modified_product_name, modified_description FROM info WHERE product_id = ?`, id)
	var mpn sql.NullString
	var md sql.NullString
	_ = row2.Scan(&mpn, &md)
	if mpn.Valid {
		p.ModifiedProductName = &mpn.String
	}
	if md.Valid {
		p.ModifiedDescription = &md.String
	}
	row3 := db.QueryRow(`SELECT modified_brand FROM brands WHERE product_id = ?`, id)
	var mb sql.NullString
	_ = row3.Scan(&mb)
	if mb.Valid {
		p.ModifiedBrand = &mb.String
	}
	row4 := db.QueryRow(`SELECT modified_listing_price, modified_sale_price, modified_discount, modified_revenue FROM finance WHERE product_id = ?`, id)
	var ml, ms, mdp, mr sql.NullString
	_ = row4.Scan(&ml, &ms, &mdp, &mr)
	if ml.Valid && ml.String != "None" && ml.String != "" {
		if v, e := strconv.ParseFloat(ml.String, 64); e == nil {
			p.ModifiedListingPrice = new(float64)
			*p.ModifiedListingPrice = v
		}
	}
	if ms.Valid && ms.String != "None" && ms.String != "" {
		if v, e := strconv.ParseFloat(ms.String, 64); e == nil {
			p.ModifiedSalePrice = new(float64)
			*p.ModifiedSalePrice = v
		}
	}
	if mdp.Valid && mdp.String != "None" && mdp.String != "" {
		if v, e := strconv.ParseFloat(mdp.String, 64); e == nil {
			p.ModifiedDiscount = new(float64)
			*p.ModifiedDiscount = v
		}
	}
	if mr.Valid && mr.String != "None" && mr.String != "" {
		if v, e := strconv.ParseFloat(mr.String, 64); e == nil {
			p.ModifiedRevenue = new(float64)
			*p.ModifiedRevenue = v
		}
	}
	row5 := db.QueryRow(`SELECT modified_last_visited FROM traffic WHERE product_id = ?`, id)
	var mlast sql.NullString
	_ = row5.Scan(&mlast)
	if mlast.Valid {
		p.ModifiedLastVisited = &mlast.String
	}
	return &p, nil
}

func UpdateModifiedFields(db *sql.DB, id string, updates map[string]interface{}) error {
	// verify product exists
	var exists int
	row := db.QueryRow(`SELECT 1 FROM info WHERE product_id = ?`, id)
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if v, ok := updates["modified_product_name"]; ok {
		_, err := tx.Exec(`UPDATE info SET modified_product_name = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_description"]; ok {
		_, err := tx.Exec(`UPDATE info SET modified_description = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_brand"]; ok {
		_, err := tx.Exec(`UPDATE brands SET modified_brand = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_listing_price"]; ok {
		_, err := tx.Exec(`UPDATE finance SET modified_listing_price = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_sale_price"]; ok {
		_, err := tx.Exec(`UPDATE finance SET modified_sale_price = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_discount"]; ok {
		_, err := tx.Exec(`UPDATE finance SET modified_discount = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_revenue"]; ok {
		_, err := tx.Exec(`UPDATE finance SET modified_revenue = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	if v, ok := updates["modified_last_visited"]; ok {
		_, err := tx.Exec(`UPDATE traffic SET modified_last_visited = ? WHERE product_id = ?`, v, id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CreateProductRequest represents the payload used to create a new product
type CreateProductRequest struct {
	ProductID    string   `json:"product_id"`
	ProductName  *string  `json:"product_name,omitempty"`
	Brand        *string  `json:"brand,omitempty"`
	ListingPrice *float64 `json:"listing_price,omitempty"`
	SalePrice    *float64 `json:"sale_price,omitempty"`
	Discount     *float64 `json:"discount,omitempty"`
	Revenue      *float64 `json:"revenue,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Rating       *float64 `json:"rating,omitempty"`
	Reviews      *float64 `json:"reviews,omitempty"`
	LastVisited  *string  `json:"last_visited,omitempty"`
}

// CreateProduct inserts a new product across the relevant tables. It validates product_id uniqueness
// and writes NULLs where optional fields are not provided.
func CreateProduct(db *sql.DB, req CreateProductRequest) error {
	if req.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	// ensure product_id not exists
	var tmp int
	row := db.QueryRow(`SELECT 1 FROM info WHERE product_id = ?`, req.ProductID)
	if err := row.Scan(&tmp); err == nil {
		return fmt.Errorf("product already exists")
	} else if err != sql.ErrNoRows {
		return err
	}
	// optionally detect duplicate by product_name (case-insensitive) if provided
	if req.ProductName != nil {
		var existingID sql.NullString
		r2 := db.QueryRow(`SELECT product_id FROM info WHERE product_name = ? COLLATE NOCASE LIMIT 1`, *req.ProductName)
		if err := r2.Scan(&existingID); err == nil {
			return fmt.Errorf("product_name already exists (product_id=%s)", existingID.String)
		} else if err != sql.ErrNoRows {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// insert into info
	var pn interface{} = nil
	var desc interface{} = nil
	if req.ProductName != nil {
		pn = *req.ProductName
	}
	if req.Description != nil {
		desc = *req.Description
	}
	if _, err := tx.Exec(`INSERT INTO info(product_name, product_id, description) VALUES(?,?,?)`, pn, req.ProductID, desc); err != nil {
		return err
	}

	// insert into brands
	var brand interface{} = nil
	if req.Brand != nil {
		brand = *req.Brand
	}
	if _, err := tx.Exec(`INSERT INTO brands(product_id, brand) VALUES(?,?)`, req.ProductID, brand); err != nil {
		return err
	}

	// insert into finance
	var lp, sp, disc, rev interface{} = nil, nil, nil, nil
	if req.ListingPrice != nil {
		lp = *req.ListingPrice
	}
	if req.SalePrice != nil {
		sp = *req.SalePrice
	}
	if req.Discount != nil {
		disc = *req.Discount
	}
	if req.Revenue != nil {
		rev = *req.Revenue
	}
	if _, err := tx.Exec(`INSERT INTO finance(product_id, listing_price, sale_price, discount, revenue) VALUES(?,?,?,?,?)`, req.ProductID, lp, sp, disc, rev); err != nil {
		return err
	}

	// insert into reviews
	var ratingStr interface{} = nil
	var reviewsStr interface{} = nil
	var realRating interface{} = nil
	var realReviews interface{} = nil
	if req.Rating != nil {
		realRating = *req.Rating
		ratingStr = fmt.Sprintf("%v", *req.Rating)
	}
	if req.Reviews != nil {
		realReviews = *req.Reviews
		reviewsStr = fmt.Sprintf("%v", *req.Reviews)
	}
	if _, err := tx.Exec(`INSERT INTO reviews(product_id, rating, reviews, real_rating, real_reviews) VALUES(?,?,?,?,?)`, req.ProductID, ratingStr, reviewsStr, realRating, realReviews); err != nil {
		return err
	}

	// insert into traffic
	var lv interface{} = nil
	if req.LastVisited != nil {
		lv = *req.LastVisited
	}
	if _, err := tx.Exec(`INSERT INTO traffic(product_id, last_visited) VALUES(?,?)`, req.ProductID, lv); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteProduct removes product rows from all tables. Returns sql.ErrNoRows if not found.
func DeleteProduct(db *sql.DB, id string) error {
	// confirm exists
	var tmp int
	row := db.QueryRow(`SELECT 1 FROM info WHERE product_id = ?`, id)
	if err := row.Scan(&tmp); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	// delete in transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM traffic WHERE product_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reviews WHERE product_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM finance WHERE product_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM brands WHERE product_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM info WHERE product_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
