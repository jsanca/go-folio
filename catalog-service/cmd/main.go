package main

import (
	"context"
	"log"

	"github.com/leatherstore/catalog-service/internal/config"
	"github.com/leatherstore/catalog-service/internal/database"
	"github.com/leatherstore/catalog-service/internal/observability"
	"github.com/leatherstore/catalog-service/internal/runtime"
	"github.com/leatherstore/catalog-service/internal/seed"
	"github.com/leatherstore/catalog-service/internal/server"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	catalogRT := runtime.NewCatalogRuntime(db)

	composite := runtime.NewComposite(catalogRT)
	defer func() {
	    if err := composite.Close(); err != nil {
	        logger.Warn("composite close", "err", err)
	    }
	}()

	seed.Run(context.Background(), catalogRT.ProductSvc, catalogRT.CatalogSvc, logger)

	srv := server.New(catalogRT, db, logger)
	logger.Info("server listening", "addr", cfg.Port)
	if err := srv.Start(ctx, cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
    	log.Fatalf("server: %v", err)
	}
}
