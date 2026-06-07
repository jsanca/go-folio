// Package service implements the application use cases for the catalog domain.
//
// Transaction ownership:
//   - The service layer owns transaction boundaries (unit of work).
//   - Repositories own SQL operations but never open or commit transactions.
//   - A repository method must never call db.BeginTx or tx.Commit.
//   - The service calls WithTx to bind repository operations to a transaction
//     it controls.
//
// Interface segregation:
//   - Each repository dependency is expressed as a small, consumer-owned interface.
//   - The same concrete *PostgresCatalogRepository satisfies multiple interfaces.
//   - The composition root (runtime package) is the only place that knows all
//     roles share one concrete object.
//
// This separation keeps business consistency rules in the service layer
// and keeps repositories focused on persistence mechanics.
package service

import (
	"context"

	"github.com/jsanca/go-folio/internal/domain"
)

// ProductUpdate holds optional fields for a partial product update.
type ProductUpdate struct {
	ProductCode      *string
	Title            *string
	Slug             *string
	ShortDescription *string
	Department       *string
	Category         *string
	PrimaryImageURL  *string
	Active           *bool
}

// CatalogService manages the product catalog: products, variants, and images.
type CatalogService interface {
	// Product operations
	CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]domain.Product, error)
	UpdateProduct(ctx context.Context, id int64, update ProductUpdate) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id int64) error

	// Variant operations
	AddVariantToProduct(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error)
	GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error)
	UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error
	ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error)

	// Image operations
	AddProductImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error)
	ListProductImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error)
	ListProductImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error)

	// Sync operations — cursor-paginated bulk reads for dotCMS integration.
	// NOTE: updatedSince on ListProductProjections filters by catalog_products.updated_at only.
	// Variant-only or image-only changes will not appear here unless the product's updated_at
	// is also updated. This is a known limitation that can be improved later.
	ListProductProjections(ctx context.Context, q domain.SyncQuery) (*domain.ProductProjectionPage, error)
	ListVariantInventory(ctx context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error)
}
