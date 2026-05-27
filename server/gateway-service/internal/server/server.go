// Package server builds the HTTP router and registers all gateway routes.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

// Server wraps the HTTP router and its dependencies.
type Server struct {
	router http.Handler
}

// New creates a Server with all gateway routes registered.
func New(rt *runtime.GatewayRuntime, logger *slog.Logger) *Server {
	r := chi.NewRouter()
	registerRoutes(r, rt, logger)
	return &Server{router: r}
}

// Start begins listening on addr and blocks until the server returns an error.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
