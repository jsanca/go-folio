package main

import (
	"context"
	"log"

	"github.com/jsanca/go-folio/gateway-service/internal/config"
	"github.com/jsanca/go-folio/gateway-service/internal/observability"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"github.com/jsanca/go-folio/gateway-service/internal/server"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger()

	rt, err := runtime.NewGatewayRuntime(context.Background(), cfg)
	if err != nil {
		log.Fatalf("init runtime: %v", err)
	}

	if rt.Auth.Permissive() {
		logger.Warn("KEYCLOAK_URL not set — auth middleware is in permissive mode, all requests pass through")
	}

	composite := runtime.NewComposite(rt)
	defer func() {
		if err := composite.Close(); err != nil {
			logger.Warn("composite close", "err", err)
		}
	}()

	srv := server.New(rt, logger)
	logger.Info("gateway listening", "addr", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
