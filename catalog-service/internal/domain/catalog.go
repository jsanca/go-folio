package domain

import (
	"errors"
	"strings"
	"time"
)

// Product is the logical/base product.
// Inventory, pricing, and SKU live on ProductVariant.
type Product struct {
	ID                int64      `json:"id"`
	ProductCode       string     `json:"productCode"`
	ExternalProductID string     `json:"externalProductId,omitempty"`
	Title             string     `json:"title"`
	Slug              string     `json:"slug"`
	ShortDescription  string     `json:"shortDescription,omitempty"`
	Description       string     `json:"description,omitempty"`
	AdditionalInfo    string     `json:"additionalInformation,omitempty"`
	Department        string     `json:"department,omitempty"`
	Category          string     `json:"category,omitempty"`
	Subcategory       string     `json:"subcategory,omitempty"`
	Tags              []string   `json:"tags,omitempty"`
	BaseSKU           string     `json:"baseSku,omitempty"`
	Active            bool       `json:"active"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	LastSyncedAt      *time.Time `json:"lastSyncedAt,omitempty"`
}

func (p *Product) Validate() error {
	if strings.TrimSpace(p.ProductCode) == "" {
		return errors.New("product code is required")
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(p.Slug) == "" {
		return errors.New("slug is required")
	}
	return nil
}

// ProductVariant is the sellable physical unit.
// SKU, pricing, stock, and color belong here.
type ProductVariant struct {
	ID                 int64       `json:"id"`
	ProductID          int64       `json:"productId"`
	SKU                string      `json:"sku"`
	ExternalVariantID  string      `json:"externalVariantId,omitempty"`
	ColorSlug          string      `json:"colorSlug,omitempty"`
	ColorName          string      `json:"colorName,omitempty"`
	PrimaryColorName   string      `json:"primaryColorName,omitempty"`
	SecondaryColorName string      `json:"secondaryColorName,omitempty"`
	PrimaryColorHex    string      `json:"primaryColorHex,omitempty"`
	SecondaryColorHex  string      `json:"secondaryColorHex,omitempty"`
	RetailPrice        Money       `json:"retailPrice"`
	SalePrice          *Money      `json:"salePrice,omitempty"`
	Currency           string      `json:"currency"`
	StockQuantity      int         `json:"stockQuantity"`
	StockStatus        StockStatus `json:"stockStatus"`
	WarehouseCode      string      `json:"warehouseCode,omitempty"`
	VariantImageURL    string      `json:"variantImageUrl,omitempty"`
	Active             bool        `json:"active"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`
	LastSyncedAt       *time.Time  `json:"lastSyncedAt,omitempty"`
}

func (v *ProductVariant) Validate() error {
	if strings.TrimSpace(v.SKU) == "" {
		return errors.New("SKU is required")
	}
	if v.ProductID <= 0 {
		return errors.New("product ID is required")
	}
	if strings.TrimSpace(v.Currency) == "" {
		return errors.New("currency is required")
	}
	if v.RetailPrice.AmountCents < 0 {
		return errors.New("retail price must not be negative")
	}
	if v.SalePrice != nil && v.SalePrice.AmountCents < 0 {
		return errors.New("sale price must not be negative")
	}
	if v.StockQuantity < 0 {
		return errors.New("stock quantity must not be negative")
	}
	if !v.StockStatus.IsValid() {
		return errors.New("invalid stock status")
	}
	return nil
}

// ProductImage represents a gallery image for a product or a specific variant.
// VariantID nil means the image belongs to the base product gallery.
type ProductImage struct {
	ID        int64      `json:"id"`
	ProductID int64      `json:"productId"`
	VariantID *int64     `json:"variantId,omitempty"`
	URL       string     `json:"url"`
	AltText   string     `json:"altText,omitempty"`
	SortOrder int        `json:"sortOrder"`
	IsMain    bool       `json:"isMain"`
	Width     *int       `json:"width,omitempty"`
	Height    *int       `json:"height,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (i *ProductImage) Validate() error {
	if i.ProductID <= 0 {
		return errors.New("product ID is required")
	}
	if strings.TrimSpace(i.URL) == "" {
		return errors.New("URL is required")
	}
	return nil
}
