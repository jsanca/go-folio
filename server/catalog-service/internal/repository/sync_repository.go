package repository

import (
	"context"

	"github.com/jsanca/go-folio/internal/domain"
)

// CatalogSyncRepository provides bulk, cursor-paginated sync queries.
// Methods are designed for large catalogs and must avoid N+1 patterns.
type CatalogSyncRepository interface {
	// ListProductProjectionPage returns one page of fully hydrated product projections
	// (product + variants + images). Results are ordered by updated_at ASC, id ASC.
	ListProductProjectionPage(ctx context.Context, q domain.SyncQuery) ([]domain.ProductProjection, bool, error)

	// ListVariantInventoryPage returns one page of lightweight variant inventory records
	// joined with the parent product code. Results are ordered by updated_at ASC, id ASC.
	ListVariantInventoryPage(ctx context.Context, q domain.SyncQuery) ([]domain.VariantInventoryRecord, bool, error)
}
