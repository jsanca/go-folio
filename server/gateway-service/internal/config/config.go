// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration for the gateway service.
type Config struct {
	Port              string // PORT env var, default: ":8090"
	CatalogURL        string // CATALOG_URL env var, default: "http://localhost:8080"
	InventoryAddr     string // INVENTORY_ADDR env var, default: "localhost:9090"
	LowStockThreshold int    // LOW_STOCK_THRESHOLD env var, default: 5
}

// Load reads configuration from environment variables and applies defaults.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8090"
	}
	catalogURL := os.Getenv("CATALOG_URL")
	if catalogURL == "" {
		catalogURL = "http://localhost:8080"
	}
	inventoryAddr := os.Getenv("INVENTORY_ADDR")
	if inventoryAddr == "" {
		inventoryAddr = "localhost:9090"
	}
	threshold := 5
	if v := os.Getenv("LOW_STOCK_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			threshold = n
		}
	}
	return Config{
		Port:              port,
		CatalogURL:        catalogURL,
		InventoryAddr:     inventoryAddr,
		LowStockThreshold: threshold,
	}
}
