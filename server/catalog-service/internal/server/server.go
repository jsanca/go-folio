// Package server builds the HTTP router and registers all routes and middleware.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jsanca/go-folio/internal/handler"
	"github.com/jsanca/go-folio/internal/observability"
	"github.com/jsanca/go-folio/internal/runtime"
)

// Server wraps the HTTP router and its dependencies.
type Server struct {
	router http.Handler
}

// New creates a Server with all routes and middleware registered.
// Middleware order is preserved: PanicRecovery → RequestLogger → Metrics.
func New(rt *runtime.CatalogRuntime, pinger observability.Pinger, logger *slog.Logger) *Server {
	metrics := observability.NewMetrics(prometheus.DefaultRegisterer)
	healthHandler := observability.NewHealthHandler(pinger, logger)
	catalogHandler := handler.NewCatalogHandler(rt.CatalogSvc)

	r := chi.NewRouter()
	r.Use(observability.PanicRecovery(logger))
	r.Use(observability.RequestLogger(logger))
	r.Use(metrics.Middleware())

	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Handle("/metrics", promhttp.Handler())

	catalogHandler.RegisterRoutes(r)

	return &Server{router: r}
}

// Start begins listening on addr and blocks until the server returns an error.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
