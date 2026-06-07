package main

import (
	"context"
	"os"

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
		logger.Error("init runtime", "err", err)
		os.Exit(1)
	}

	if rt.Auth.Permissive() {
		logger.Warn("permissive auth is active — all requests are unauthenticated; DO NOT use in production")
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
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}
