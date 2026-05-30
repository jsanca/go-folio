// Package runtime wires all repositories and services together.
// It is the composition root for the inventory service.
package runtime

import (
	"database/sql"
	"log/slog"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/inventory-service/internal/repository"
	"github.com/jsanca/go-folio/inventory-service/internal/service"
)

// InventoryRuntime holds all services for the inventory domain.
// The caller retains ownership of the DB connection — InventoryRuntime does not close it.
type InventoryRuntime struct {
	Svc  invpb.InventoryServiceServer
	Repo repository.Repository
}

// NewInventoryRuntime wires the inventory repository and service.
func NewInventoryRuntime(db *sql.DB, logger *slog.Logger) *InventoryRuntime {
	repo := repository.NewSQLiteRepository(db)
	svc := service.NewService(db, repo, logger)
	return &InventoryRuntime{
		Svc:  svc,
		Repo: repo,
	}
}

// Close releases any resources owned by InventoryRuntime.
// Implements io.Closer. Currently a no-op; reserved for future gRPC client teardown.
func (rt *InventoryRuntime) Close() error {
	return nil
}
