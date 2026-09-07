package models

import "time"

// User is an account that can sign in to the application.
// Password should never be rendered to templates; it keeps the hash only
// in memory between login checks.
type User struct {
	ID         int64
	Name       string
	Email      string
	Phone      string
	Password   string
	Role       string // "customer" or "admin"
	Address    string
	CreatedAt  time.Time
	OrderCount int
	TotalSpent int64
}

// Role constants used across the application.
const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)