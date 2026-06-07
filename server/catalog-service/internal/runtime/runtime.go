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
// The four-argument NewCatalogService call is intentional: it is the explicit
// cost of interface segregation at this layer.
func NewCatalogRuntime(db *sql.DB) *CatalogRuntime {
	catalogRepo := repository.NewPostgresCatalogRepository(db)
	return &CatalogRuntime{
		CatalogSvc: service.NewCatalogService(db, catalogRepo, catalogRepo, catalogRepo, catalogRepo),
	}
}

// Close releases any resources owned by CatalogRuntime.
// Implements io.Closer. Currently a no-op; reserved for future gRPC client teardown.
func (rt *CatalogRuntime) Close() error {
	return nil
}
