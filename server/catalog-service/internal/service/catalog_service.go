package service

import (
	"context"

	"github.com/jsanca/go-folio/internal/domain"
)

// CatalogService manages the product catalog: products, variants, and images.
type CatalogService interface {
	// Product operations
	CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]domain.Product, error)

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
