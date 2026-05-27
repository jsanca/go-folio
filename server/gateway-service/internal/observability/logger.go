// Package observability provides structured logging for the gateway service.
package observability

import (
	"log/slog"
	"os"
)

// NewLogger returns a JSON-formatted slog.Logger writing to stdout at Info level.
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
