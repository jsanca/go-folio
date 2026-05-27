package domain

// StockStatus represents the availability state of a product variant.
type StockStatus string

const (
	// StockStatusInStock indicates sufficient available stock.
	StockStatusInStock StockStatus = "IN_STOCK"
	// StockStatusLowStock indicates stock is available but running low.
	StockStatusLowStock StockStatus = "LOW_STOCK"
	// StockStatusOutOfStock indicates no available stock.
	StockStatusOutOfStock StockStatus = "OUT_OF_STOCK"
)

// IsValid reports whether the StockStatus value is one of the defined constants.
func (s StockStatus) IsValid() bool {
	switch s {
	case StockStatusInStock, StockStatusLowStock, StockStatusOutOfStock:
		return true
	}
	return false
}

// Money represents a monetary amount as integer cents to avoid floating-point issues.
type Money struct {
	AmountCents int64 `json:"amountCents"`
}
