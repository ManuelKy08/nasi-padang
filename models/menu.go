package models

import "time"

// MenuItem is a single sellable dish.
// Price is stored as the lowest unit of Rupiah (integer) to avoid float drift.
type MenuItem struct {
	ID           int64
	CategoryID   int64
	CategoryName string
	Name         string
	Slug         string
	Description  string
	Price        int64
	Image        string
	Available    bool
	CreatedAt    time.Time
	IsFavorite   bool
	// Quantity is populated when a cart joins the item.
	Quantity int
	// TotalQty is populated by popularity queries.
	TotalQty int
}