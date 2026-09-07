package models

import "time"

// CartItem is a row of a user's cart enriched with its menu item.
type CartItem struct {
	ID         int64
	UserID     int64
	MenuItemID int64
	Quantity   int
	CreatedAt  time.Time
	Menu       MenuItem
}