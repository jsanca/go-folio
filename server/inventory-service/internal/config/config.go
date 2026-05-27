// Package config loads runtime configuration from environment variables.
package config

import "os"

// Config holds all runtime configuration for the inventory service.
type Config struct {
	DBPath string // DB_PATH env var, default: "./inventory.db"
	Port   string // PORT env var, default: ":9090"
}

// Load reads configuration from environment variables and applies defaults.
func Load() Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./inventory.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = ":9090"
	}
	return Config{DBPath: dbPath, Port: port}
}
