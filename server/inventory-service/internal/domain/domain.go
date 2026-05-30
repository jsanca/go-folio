// Package domain contains the core types for inventory management.
package domain

// Stock represents the current stock levels for a single SKU.
type Stock struct {
	SKU       string
	Available int32
	Reserved  int32
}

// Reservation represents a held quantity of a SKU against an order.
type Reservation struct {
	ID       string
	SKU      string
	Quantity int32
	OrderID  string
}
