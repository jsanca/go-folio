// Package runtime wires all repositories and services together.
// It is the composition root for the catalog service.
package runtime

import (
	"database/sql"

	"github.com/jsanca/go-folio/internal/repository"
	"github.com/jsanca/go-folio/internal/service"
)

// CatalogRuntime holds all services for the catalog domain.
// The caller retains ownership of the DB connection — CatalogRuntime does not close it.
type CatalogRuntime struct {
	CatalogSvc service.CatalogService
}

// NewCatalogRuntime wires all catalog repositories and services.
// The same *PostgresCatalogRepository satisfies all seven consumer-owned
// interfaces (ProductReader, ProductWriter, VariantReader, VariantWriter,
// ImageReader, ImageWriter, SyncReader). Passing it once per role keeps each
// interface small and each service dependency explicit. This is the only place
// in the codebase that knows all roles share one concrete object.
func NewCatalogRuntime(db *sql.DB) *CatalogRuntime {
	catalogRepo := repository.NewPostgresCatalogRepository(db)
	return &CatalogRuntime{
		CatalogSvc: service.NewCatalogService(
			db,
			catalogRepo, // ProductReader
			catalogRepo, // ProductWriter
			catalogRepo, // VariantReader
			catalogRepo, // VariantWriter
			catalogRepo, // ImageReader
			catalogRepo, // ImageWriter
			catalogRepo, // SyncReader
		),
	}
}

// Close releases any resources owned by CatalogRuntime.
// Implements io.Closer. Currently a no-op; reserved for future gRPC client teardown.
func (rt *CatalogRuntime) Close() error {
	return nil
}
