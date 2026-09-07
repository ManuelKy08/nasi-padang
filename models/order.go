package models

import "time"

// Order types.
const (
	OrderTypeDineIn  = "dine_in"
	OrderTypeTakeAway = "take_away"
	OrderTypeDelivery = "delivery"
)

// Payment methods (simulated, no real gateway).
const (
	PayCash        = "cash"
	PayBank        = "bank_transfer"
	PayEWallet     = "ewallet"
)

// Order statuses, in lifecycle order for dine-in / takeaway / delivery.
const (
	StatusPending    = "pending"    // PESANAN DIBUAT
	StatusProcessing = "processing" // DIPROSES
	StatusCooking    = "cooking"    // DIMASAK
	StatusReady      = "ready"      // SIAP
	StatusDelivering = "delivering" // DIANTAR (delivery only)
	StatusServed     = "served"     // DISAJIKAN (dine-in only)
	StatusCompleted  = "completed"  // SELESAI
	StatusCancelled  = "cancelled"  // BATAL
)

// Order is a placed order with its items.
type Order struct {
	ID            int64
	UserID        int64
	UserName      string
	OrderNumber   string
	OrderType     string
	Subtotal      int64
	DeliveryFee   int64
	Total         int64
	PaymentMethod string
	Status        string
	Address       string
	TableNumber   string
	Notes         string
	CreatedAt     time.Time
	Items         []OrderItem
}

// OrderTypeLabel returns the Indonesian name for an order type.
func OrderTypeLabel(t string) string {
	switch t {
	case OrderTypeDineIn:
		return "Makan di Tempat"
	case OrderTypeTakeAway:
		return "Take Away"
	case OrderTypeDelivery:
		return "Delivery"
	}
	return t
}

// PaymentLabel returns the Indonesian name for a payment method.
func PaymentLabel(m string) string {
	switch m {
	case PayCash:
		return "Cash"
	case PayBank:
		return "Bank Transfer"
	case PayEWallet:
		return "E-Wallet"
	}
	return m
}

// StatusLabel returns the Indonesian status name.
func StatusLabel(s string) string {
	switch s {
	case StatusPending:
		return "Pesanan Dibuat"
	case StatusProcessing:
		return "Diproses"
	case StatusCooking:
		return "Sedang Dimasak"
	case StatusReady:
		return "Siap"
	case StatusDelivering:
		return "Diantar"
	case StatusServed:
		return "Disajikan"
	case StatusCompleted:
		return "Selesai"
	case StatusCancelled:
		return "Dibatalkan"
	}
	return s
}