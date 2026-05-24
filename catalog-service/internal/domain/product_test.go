package domain

import (
	"testing"
)

func validProduct() LeatherProduct {
	return LeatherProduct{
		SKU:           "LTH-001",
		Title:         "Bolso de Cuero Premium",
		Slug:          "bolso-cuero-premium",
		Currency:      "USD",
		RetailPrice:   Money{AmountCents: 19999},
		StockQuantity: 10,
		StockStatus:   StockStatusInStock,
	}
}

func TestValidate_ValidProduct(t *testing.T) {
	p := validProduct()
	if err := p.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_OptionalSalePrice(t *testing.T) {
	p := validProduct()
	sale := Money{AmountCents: 14999}
	p.SalePrice = &sale
	if err := p.Validate(); err != nil {
		t.Errorf("expected no error with valid sale price, got: %v", err)
	}
}

func TestValidate_MissingSKU(t *testing.T) {
	p := validProduct()
	p.SKU = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing SKU")
	}
}

func TestValidate_MissingTitle(t *testing.T) {
	p := validProduct()
	p.Title = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing title")
	}
}

func TestValidate_MissingSlug(t *testing.T) {
	p := validProduct()
	p.Slug = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing slug")
	}
}

func TestValidate_MissingCurrency(t *testing.T) {
	p := validProduct()
	p.Currency = ""
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing currency")
	}
}

func TestValidate_NegativeRetailPrice(t *testing.T) {
	p := validProduct()
	p.RetailPrice = Money{AmountCents: -1}
	if err := p.Validate(); err == nil {
		t.Error("expected error for negative retail price")
	}
}

func TestValidate_NegativeSalePrice(t *testing.T) {
	p := validProduct()
	sale := Money{AmountCents: -5}
	p.SalePrice = &sale
	if err := p.Validate(); err == nil {
		t.Error("expected error for negative sale price")
	}
}

func TestValidate_ZeroPriceIsValid(t *testing.T) {
	p := validProduct()
	p.RetailPrice = Money{AmountCents: 0}
	if err := p.Validate(); err != nil {
		t.Errorf("expected zero price to be valid, got: %v", err)
	}
}

func TestValidate_NegativeStockQuantity(t *testing.T) {
	p := validProduct()
	p.StockQuantity = -1
	if err := p.Validate(); err == nil {
		t.Error("expected error for negative stock quantity")
	}
}

func TestValidate_InvalidStockStatus(t *testing.T) {
	p := validProduct()
	p.StockStatus = "UNKNOWN"
	if err := p.Validate(); err == nil {
		t.Error("expected error for invalid stock status")
	}
}

func TestStockStatus_IsValid(t *testing.T) {
	valid := []StockStatus{StockStatusInStock, StockStatusLowStock, StockStatusOutOfStock}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []StockStatus{"", "UNKNOWN", "in_stock"}
	for _, s := range invalid {
		if s.IsValid() {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
