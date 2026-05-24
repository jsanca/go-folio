package domain

import (
	"testing"
)

// ── Product validation ────────────────────────────────────────────────────────

func TestProduct_ValidatesWithoutSKUOrPriceOrStock(t *testing.T) {
	p := &Product{
		ProductCode: "BM-02",
		Title:       "Billetera de Colores",
		Slug:        "billetera-de-colores",
		Active:      true,
	}
	if err := p.Validate(); err != nil {
		t.Errorf("expected valid product, got: %v", err)
	}
}

func TestProduct_RequiresProductCode(t *testing.T) {
	p := &Product{Title: "T", Slug: "s"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing product code")
	}
}

func TestProduct_RequiresTitle(t *testing.T) {
	p := &Product{ProductCode: "X", Slug: "s"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing title")
	}
}

func TestProduct_RequiresSlug(t *testing.T) {
	p := &Product{ProductCode: "X", Title: "T"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for missing slug")
	}
}

// ── ProductVariant validation ─────────────────────────────────────────────────

func TestProductVariant_RequiresSKU(t *testing.T) {
	v := validVariant()
	v.SKU = ""
	if err := v.Validate(); err == nil {
		t.Error("expected error for missing SKU")
	}
}

func TestProductVariant_RequiresProductID(t *testing.T) {
	v := validVariant()
	v.ProductID = 0
	if err := v.Validate(); err == nil {
		t.Error("expected error for missing product ID")
	}
}

func TestProductVariant_RequiresCurrency(t *testing.T) {
	v := validVariant()
	v.Currency = ""
	if err := v.Validate(); err == nil {
		t.Error("expected error for missing currency")
	}
}

func TestProductVariant_RequiresValidStockStatus(t *testing.T) {
	v := validVariant()
	v.StockStatus = "BOGUS"
	if err := v.Validate(); err == nil {
		t.Error("expected error for invalid stock status")
	}
}

func TestProductVariant_SupportsCRCPriceInAmountCents(t *testing.T) {
	v := validVariant()
	v.Currency = "CRC"
	v.RetailPrice = Money{AmountCents: 1500000} // ₡15 000.00
	sale := Money{AmountCents: 1200000}
	v.SalePrice = &sale

	if err := v.Validate(); err != nil {
		t.Errorf("expected valid CRC variant, got: %v", err)
	}
}

func TestProductVariant_SupportsColorSlugNameAndHex(t *testing.T) {
	v := validVariant()
	v.ColorSlug = "cafe"
	v.PrimaryColorName = "Café"
	v.PrimaryColorHex = "#26110a"

	if err := v.Validate(); err != nil {
		t.Errorf("expected valid colored variant, got: %v", err)
	}
	if v.ColorSlug != "cafe" {
		t.Errorf("unexpected color slug: %s", v.ColorSlug)
	}
	if v.PrimaryColorHex != "#26110a" {
		t.Errorf("unexpected hex: %s", v.PrimaryColorHex)
	}
}

func TestProductVariant_RejectsNegativeRetailPrice(t *testing.T) {
	v := validVariant()
	v.RetailPrice = Money{AmountCents: -1}
	if err := v.Validate(); err == nil {
		t.Error("expected error for negative retail price")
	}
}

func TestProductVariant_RejectsNegativeSalePrice(t *testing.T) {
	v := validVariant()
	sale := Money{AmountCents: -1}
	v.SalePrice = &sale
	if err := v.Validate(); err == nil {
		t.Error("expected error for negative sale price")
	}
}

func TestProductVariant_RejectsNegativeStockQuantity(t *testing.T) {
	v := validVariant()
	v.StockQuantity = -5
	if err := v.Validate(); err == nil {
		t.Error("expected error for negative stock quantity")
	}
}

// ── ProductImage validation ───────────────────────────────────────────────────

func TestProductImage_ValidProductLevel(t *testing.T) {
	img := &ProductImage{ProductID: 1, URL: "https://example.com/img.jpg"}
	if err := img.Validate(); err != nil {
		t.Errorf("expected valid product image, got: %v", err)
	}
}

func TestProductImage_ValidVariantLevel(t *testing.T) {
	vid := int64(5)
	img := &ProductImage{ProductID: 1, VariantID: &vid, URL: "https://example.com/img.jpg"}
	if err := img.Validate(); err != nil {
		t.Errorf("expected valid variant image, got: %v", err)
	}
}

func TestProductImage_RequiresProductID(t *testing.T) {
	img := &ProductImage{URL: "https://example.com/img.jpg"}
	if err := img.Validate(); err == nil {
		t.Error("expected error for missing product ID")
	}
}

func TestProductImage_RequiresURL(t *testing.T) {
	img := &ProductImage{ProductID: 1}
	if err := img.Validate(); err == nil {
		t.Error("expected error for missing URL")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func validVariant() *ProductVariant {
	return &ProductVariant{
		ProductID:     1,
		SKU:           "BM-02-COL-CO-CA",
		Currency:      "CRC",
		RetailPrice:   Money{AmountCents: 1500000},
		StockQuantity: 10,
		StockStatus:   StockStatusInStock,
	}
}
