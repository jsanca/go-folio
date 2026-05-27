package repository

import (
	"context"

	"github.com/jsanca/go-folio/internal/domain"
)

// CatalogProductRepository manages the catalog_products table.
type CatalogProductRepository interface {
	CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]domain.Product, error)
}

// ProductVariantRepository manages the product_variants table.
type ProductVariantRepository interface {
	AddVariant(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error)
	GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error)
	GetVariantByID(ctx context.Context, id int64) (*domain.ProductVariant, error)
	UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error
	ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error)
}

// ProductImageRepository manages the product_images table.
type ProductImageRepository interface {
	AddImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error)
	ListImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error)
	ListImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error)
}
