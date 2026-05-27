// Package config loads runtime configuration from environment variables.
package config

import "os"

// Config holds all runtime configuration for the catalog service.
type Config struct {
	DBPath string // DB_PATH env var, default: "./leatherstore.db"
	Port   string // PORT env var, default: ":8080"
}

// Load reads configuration from environment variables and applies defaults.
func Load() Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./leatherstore.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	return Config{DBPath: dbPath, Port: port}
}
