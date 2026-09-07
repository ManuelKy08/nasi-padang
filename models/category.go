package models

// Category groups menu items (NASI, LAUK, SAYUR, SAMBAL, MINUMAN, PAKET).
type Category struct {
	ID   int64
	Name string
	Slug string
}