// Package config loads runtime configuration from environment variables.
package config

import "os"

// Config holds all runtime configuration for the inventory service.
type Config struct {
	DatabaseURL string // DATABASE_URL env var
	Port        string // PORT env var, default: ":9090"
}

// Load reads configuration from environment variables and applies defaults.
func Load() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://folio:folio@localhost:5432/folio_inventory?sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = ":9090"
	}
	return Config{DatabaseURL: dbURL, Port: port}
}
