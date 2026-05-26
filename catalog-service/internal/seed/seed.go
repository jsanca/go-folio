// Package seed populates the database with initial data when it is empty.
package seed

import (
	"context"
	"log"
	"log/slog"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/service"
)

// Run seeds both product and catalog data if the database is empty.
// It is a no-op if data already exists.
func Run(ctx context.Context, productSvc service.ProductService, catalogSvc service.CatalogService, logger *slog.Logger) {
	seedProducts(ctx, productSvc)
	seedCatalog(ctx, catalogSvc, logger)
}

func seedProducts(ctx context.Context, svc service.ProductService) {
	existing, err := svc.ListProducts(ctx)
	if err != nil {
		log.Printf("seed: check existing products: %v", err)
		return
	}
	if len(existing) > 0 {
		return
	}

	sale := func(cents int64) *domain.Money { return &domain.Money{AmountCents: cents} }

	products := []*domain.LeatherProduct{
		{
			SKU: "BAG-TOTE-001", ExternalSystemID: "SAP-1001",
			Title: "Classic Leather Tote", Slug: "classic-leather-tote",
			ShortDescription: "Spacious handcrafted tote bag.",
			Description:      "Full-grain leather tote with brass hardware.",
			Category: "Bags", Tags: []string{"tote", "leather", "handmade"},
			MainImageURL:  "https://example.com/images/classic-tote.jpg",
			RetailPrice:   domain.Money{AmountCents: 24900}, SalePrice: sale(19900), Currency: "USD",
			StockQuantity: 15, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-001", Active: true,
		},
		{
			SKU: "WAL-BIF-001", ExternalSystemID: "SAP-1002",
			Title: "Slim Bifold Wallet", Slug: "slim-bifold-wallet",
			ShortDescription: "Minimalist wallet in full-grain leather.",
			Description:      "Six card slots, one bill compartment. Fits in any pocket.",
			Category: "Wallets", Tags: []string{"wallet", "slim", "bifold"},
			MainImageURL:  "https://example.com/images/bifold-wallet.jpg",
			RetailPrice:   domain.Money{AmountCents: 8900}, Currency: "USD",
			StockQuantity: 40, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-001", Active: true,
		},
		{
			SKU: "BEL-CLB-001", ExternalSystemID: "SAP-1003",
			Title: "Classic Dress Belt", Slug: "classic-dress-belt",
			ShortDescription: "Full-grain leather belt with polished buckle.",
			Description:      "Available in black and brown. Width: 35 mm.",
			Category: "Belts", Tags: []string{"belt", "dress", "formal"},
			MainImageURL:  "https://example.com/images/dress-belt.jpg",
			RetailPrice:   domain.Money{AmountCents: 6500}, Currency: "USD",
			StockQuantity: 25, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-002", Active: true,
		},
		{
			SKU: "BAG-MSG-001", ExternalSystemID: "SAP-1004",
			Title: "Leather Messenger Bag", Slug: "leather-messenger-bag",
			ShortDescription: "Professional messenger with padded laptop sleeve.",
			Description:      "Fits 15-inch laptops. Adjustable shoulder strap.",
			Category: "Bags", Tags: []string{"messenger", "laptop", "professional"},
			MainImageURL:  "https://example.com/images/messenger-bag.jpg",
			RetailPrice:   domain.Money{AmountCents: 31500}, SalePrice: sale(26900), Currency: "USD",
			StockQuantity: 8, StockStatus: domain.StockStatusLowStock,
			WarehouseCode: "WH-001", Active: true,
		},
		{
			SKU: "ACC-KCH-001", ExternalSystemID: "SAP-1005",
			Title: "Leather Key Chain", Slug: "leather-key-chain",
			ShortDescription: "Simple and elegant keychain.",
			Description:      "Vegetable-tanned leather with brass ring.",
			Category: "Accessories", Tags: []string{"keychain", "gift", "small"},
			MainImageURL:  "https://example.com/images/key-chain.jpg",
			RetailPrice:   domain.Money{AmountCents: 1500}, Currency: "USD",
			StockQuantity: 100, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-003", Active: true,
		},
		{
			SKU: "BAG-BCK-001", ExternalSystemID: "SAP-1006",
			Title: "Leather Backpack", Slug: "leather-backpack",
			ShortDescription: "Rugged backpack for daily use.",
			Description:      "Three compartments. Padded back panel.",
			Category: "Bags", Tags: []string{"backpack", "travel", "rugged"},
			MainImageURL:  "https://example.com/images/backpack.jpg",
			RetailPrice:   domain.Money{AmountCents: 42000}, Currency: "USD",
			StockQuantity: 0, StockStatus: domain.StockStatusOutOfStock,
			WarehouseCode: "WH-001", Active: true,
		},
		{
			SKU: "WAL-ZIP-001", ExternalSystemID: "SAP-1007",
			Title: "Zip-Around Wallet", Slug: "zip-around-wallet",
			ShortDescription: "Full-zip wallet with coin pocket.",
			Description:      "Twelve card slots and a coin zipper compartment.",
			Category: "Wallets", Tags: []string{"wallet", "zip", "travel"},
			MainImageURL:  "https://example.com/images/zip-wallet.jpg",
			RetailPrice:   domain.Money{AmountCents: 11200}, SalePrice: sale(9500), Currency: "USD",
			StockQuantity: 20, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-002", Active: true,
		},
		{
			SKU: "ACC-CAR-001", ExternalSystemID: "SAP-1008",
			Title: "Leather Card Holder", Slug: "leather-card-holder",
			ShortDescription: "Ultra-thin card holder for 4 cards.",
			Description:      "Perfect for minimalists. Vegetable-tanned hide.",
			Category: "Accessories", Tags: []string{"cardholder", "minimalist", "slim"},
			MainImageURL:  "https://example.com/images/card-holder.jpg",
			RetailPrice:   domain.Money{AmountCents: 4500}, Currency: "USD",
			StockQuantity: 60, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-003", Active: true,
		},
		{
			SKU: "BAG-CLT-001", ExternalSystemID: "SAP-1009",
			Title: "Leather Clutch", Slug: "leather-clutch",
			ShortDescription: "Elegant evening clutch.",
			Description:      "Magnetic clasp, interior card slots, wrist strap.",
			Category: "Bags", Tags: []string{"clutch", "evening", "elegant"},
			MainImageURL:  "https://example.com/images/clutch.jpg",
			RetailPrice:   domain.Money{AmountCents: 16800}, SalePrice: sale(13900), Currency: "USD",
			StockQuantity: 12, StockStatus: domain.StockStatusInStock,
			WarehouseCode: "WH-002", Active: true,
		},
		{
			SKU: "BEL-CSL-001", ExternalSystemID: "SAP-1010",
			Title: "Casual Braided Belt", Slug: "casual-braided-belt",
			ShortDescription: "Braided leather belt for casual wear.",
			Description:      "Handwoven strips. Adjustable length. No holes needed.",
			Category: "Belts", Tags: []string{"belt", "braided", "casual"},
			MainImageURL:  "https://example.com/images/braided-belt.jpg",
			RetailPrice:   domain.Money{AmountCents: 5500}, Currency: "USD",
			StockQuantity: 3, StockStatus: domain.StockStatusLowStock,
			WarehouseCode: "WH-002", Active: false,
		},
	}

	for _, p := range products {
		if _, err := svc.CreateProduct(ctx, p); err != nil {
			log.Printf("seed: create %s: %v", p.SKU, err)
		}
	}
	log.Printf("seed: products seeded count=%d", len(products))
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
