package repository

import (
	"context"
	"database/sql"

	"github.com/jsanca/go-folio/internal/domain"
)

// ProductReader defines non-transactional product read operations.
// Used by service methods that only query product data without acquiring locks.
type ProductReader interface {
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context) ([]domain.Product, error)
}

// ProductWriter defines product mutation operations and transaction binding.
// GetProductByIDForUpdate is included here because it is only used inside
// write transactions (SELECT … FOR UPDATE) and always precedes a mutating
// operation. The service calls WithTx to obtain a transaction-scoped writer.
type ProductWriter interface {
	GetProductByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error)
	CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id int64, p *domain.Product) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	// WithTx returns a transaction-scoped ProductWriter bound to tx.
	// The caller (service layer) owns the transaction lifecycle.
	WithTx(tx *sql.Tx) ProductWriter
}

// VariantReader defines read-only variant access.
type VariantReader interface {
	GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error)
	GetVariantByID(ctx context.Context, id int64) (*domain.ProductVariant, error)
	ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error)
}

// VariantWriter defines variant mutation operations.
type VariantWriter interface {
	AddVariant(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error)
	UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error
}

// ImageReader defines read-only image access.
type ImageReader interface {
	ListImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error)
	ListImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error)
}

// ImageWriter defines image mutation operations.
type ImageWriter interface {
	AddImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error)
}

// SyncReader defines bulk, cursor-paginated projection queries for external integration.
// Methods are designed for large catalogs and must avoid N+1 patterns.
type SyncReader interface {
	// ListProductProjectionPage returns one page of fully hydrated product projections
	// (product + variants + images). Results are ordered by updated_at ASC, id ASC.
	ListProductProjectionPage(ctx context.Context, q domain.SyncQuery) ([]domain.ProductProjection, bool, error)

	// ListVariantInventoryPage returns one page of lightweight variant inventory records
	// joined with the parent product code. Results are ordered by updated_at ASC, id ASC.
	ListVariantInventoryPage(ctx context.Context, q domain.SyncQuery) ([]domain.VariantInventoryRecord, bool, error)
}
