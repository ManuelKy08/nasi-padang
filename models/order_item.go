package models

// OrderItem is a line on an order.
type OrderItem struct {
	ID         int64
	OrderID    int64
	MenuItemID int64
	MenuName   string
	Quantity   int
	Price      int64
}