// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration for the gateway service.
type Config struct {
	Port               string // PORT env var, default: ":8090"
	CatalogURL         string // CATALOG_URL env var, default: "http://localhost:8080"
	InventoryAddr      string // INVENTORY_ADDR env var, default: "localhost:9090"
	LowStockThreshold  int    // LOW_STOCK_THRESHOLD env var, default: 5
	KeycloakURL        string // KEYCLOAK_URL env var, default: "http://localhost:8180"
	KeycloakRealm      string // KEYCLOAK_REALM env var, default: "folio"
	CORSAllowedOrigins string // CORS_ALLOWED_ORIGINS env var, comma-separated; default: localhost:3000,3001
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
	// KeycloakURL intentionally has no fallback default. An empty value puts
	// the gateway into permissive mode (no JWT validation), which is the safe
	// default for local development without Docker. Set KEYCLOAK_URL explicitly
	// (e.g. http://localhost:8180 or http://keycloak:8080 in Docker) to enable
	// token validation.
	keycloakURL := os.Getenv("KEYCLOAK_URL")
	keycloakRealm := os.Getenv("KEYCLOAK_REALM")
	if keycloakRealm == "" {
		keycloakRealm = "folio"
	}
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://localhost:3001"
	}
	return Config{
		Port:               port,
		CatalogURL:         catalogURL,
		InventoryAddr:      inventoryAddr,
		LowStockThreshold:  threshold,
		KeycloakURL:        keycloakURL,
		KeycloakRealm:      keycloakRealm,
		CORSAllowedOrigins: corsOrigins,
	}
}
