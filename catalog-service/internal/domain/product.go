package domain

import (
	"errors"
	"strings"
	"time"
)

type StockStatus string

const (
	StockStatusInStock    StockStatus = "IN_STOCK"
	StockStatusLowStock   StockStatus = "LOW_STOCK"
	StockStatusOutOfStock StockStatus = "OUT_OF_STOCK"
)

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

type LeatherProduct struct {
	ID               int64       `json:"id"`
	SKU              string      `json:"sku"`
	ExternalSystemID string      `json:"externalSystemId"`
	Title            string      `json:"title"`
	Slug             string      `json:"slug"`
	ShortDescription string      `json:"shortDescription"`
	Description      string      `json:"description"`
	Category         string      `json:"category"`
	Tags             []string    `json:"tags"`
	MainImageURL     string      `json:"mainImageUrl"`
	RetailPrice      Money       `json:"retailPrice"`
	SalePrice        *Money      `json:"salePrice,omitempty"`
	Currency         string      `json:"currency"`
	StockQuantity    int         `json:"stockQuantity"`
	StockStatus      StockStatus `json:"stockStatus"`
	WarehouseCode    string      `json:"warehouseCode"`
	Active           bool        `json:"active"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	LastSyncedAt     *time.Time  `json:"lastSyncedAt,omitempty"`
}

func (p *LeatherProduct) Validate() error {
	if strings.TrimSpace(p.SKU) == "" {
		return errors.New("SKU is required")
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(p.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return errors.New("currency is required")
	}
	if p.RetailPrice.AmountCents < 0 {
		return errors.New("retail price must not be negative")
	}
	if p.SalePrice != nil && p.SalePrice.AmountCents < 0 {
		return errors.New("sale price must not be negative")
	}
	if p.StockQuantity < 0 {
		return errors.New("stock quantity must not be negative")
	}
	if !p.StockStatus.IsValid() {
		return errors.New("invalid stock status")
	}
	return nil
}
