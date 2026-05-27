// Package seed populates the database with initial data when it is empty.
package seed

import (
	"context"
	"log"
	"log/slog"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/service"
)

// Run seeds catalog data if the database is empty.
// It is a no-op if data already exists.
func Run(ctx context.Context, catalogSvc service.CatalogService, logger *slog.Logger) {
	seedCatalog(ctx, catalogSvc, logger)
}

func seedCatalog(ctx context.Context, svc service.CatalogService, logger *slog.Logger) {
	existing, err := svc.ListProducts(ctx)
	if err != nil || len(existing) > 0 {
		return
	}

	product, err := svc.CreateProduct(ctx, &domain.Product{
		ProductCode:       "BM-02",
		ExternalProductID: "1984",
		Title:             "Billetera de Colores",
		Slug:              "billetera-de-colores",
		ShortDescription:  "Billetera de cuero en múltiples colores.",
		Department:        "ELLAS",
		Category:          "Accesorios",
		Subcategory:       "Billeteras",
		BaseSKU:           "BM-02-Colores",
		Active:            true,
	})
	if err != nil {
		log.Printf("seed catalog: create product: %v", err)
		return
	}

	type variantSeed struct {
		sku, slug, name, hex string
		price                int64
		stock                int
	}
	variants := []variantSeed{
		{"BM-02-COL-CO-NUE", "nuez", "Nuez", "#7b3500", 2439000, 13},
		{"BM-02-COL-CO-NE", "negro", "Negro", "#000000", 2439000, 8},
		{"BM-02-COL-CO-MI", "miel", "Miel", "#f5a52c", 2439000, 5},
		{"BM-02-COL-CO-CA", "cafe", "Café", "#26110a", 2439000, 12},
	}
	for _, v := range variants {
		if _, err := svc.AddVariantToProduct(ctx, &domain.ProductVariant{
			ProductID:        product.ID,
			SKU:              v.sku,
			ColorSlug:        v.slug,
			ColorName:        v.name,
			PrimaryColorName: v.name,
			PrimaryColorHex:  v.hex,
			RetailPrice:      domain.Money{AmountCents: v.price},
			Currency:         "CRC",
			StockQuantity:    v.stock,
			StockStatus:      domain.StockStatusInStock,
			WarehouseCode:    "WH-001",
			Active:           true,
		}); err != nil {
			log.Printf("seed catalog: add variant %s: %v", v.sku, err)
		}
	}
	logger.Info("catalog seeded", "product", product.ProductCode, "variants", len(variants))
}
