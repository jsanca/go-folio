// Package seed populates the stock table with initial SKU data when it is empty.
package seed

import (
	"context"
	"log/slog"

	"github.com/jsanca/go-folio/inventory-service/internal/inventory"
)

// skus mirrors the SKUs seeded in catalog-service (both the 10 LeatherProduct
// base SKUs and the 4 BM-02 variants). Quantities match catalog-service/seed/seed.go.
var skus = []struct {
	sku       string
	available int32
}{
	{"BAG-TOTE-001", 15},
	{"WAL-BIF-001", 40},
	{"BEL-CLB-001", 25},
	{"BAG-MSG-001", 8},
	{"ACC-KCH-001", 100},
	{"BAG-BCK-001", 0},
	{"WAL-ZIP-001", 20},
	{"ACC-CAR-001", 60},
	{"BAG-CLT-001", 12},
	{"BEL-CSL-001", 3},
	// BM-02 Billetera de Colores variants
	{"BM-02-COL-CO-NUE", 13},
	{"BM-02-COL-CO-NE", 8},
	{"BM-02-COL-CO-MI", 5},
	{"BM-02-COL-CO-CA", 12},
}

// Run seeds the stock table with catalog SKUs if it is empty.
// It is a no-op if any stock records already exist.
func Run(ctx context.Context, repo inventory.Repository, logger *slog.Logger) {
	has, err := repo.HasAnyStock(ctx)
	if err != nil || has {
		return
	}
	for _, s := range skus {
		if err := repo.SeedSKU(ctx, s.sku, s.available); err != nil {
			logger.Error("seed sku", "sku", s.sku, "error", err)
		}
	}
	logger.Info("inventory seeded", "count", len(skus))
}
