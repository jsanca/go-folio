package repository

import (
	"context"
	"database/sql"

	"github.com/jsanca/go-folio/internal/domain"
)

// CatalogProductRepository manages the catalog_products table.
// Implementations must not manage transaction boundaries — callers own
// the unit of work and may pass a tx-scoped repository via WithTx.
type CatalogProductRepository interface {
	CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	// GetProductByIDForUpdate fetches a product and acquires a row lock (SELECT … FOR UPDATE).
	// It must be called inside a transaction to have effect.
	GetProductByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]domain.Product, error)
	UpdateProduct(ctx context.Context, id int64, p *domain.Product) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	// WithTx returns a transaction-scoped CatalogProductRepository. The caller is
	// responsible for Commit/Rollback on the underlying *sql.Tx.
	WithTx(tx *sql.Tx) CatalogProductRepository
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
