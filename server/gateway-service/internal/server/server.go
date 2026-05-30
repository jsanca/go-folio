// Package server builds the HTTP router and registers all gateway routes.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"github.com/rs/cors"
)

// Server wraps the HTTP router and its dependencies.
type Server struct {
	router http.Handler
}

// New creates a Server with all gateway routes registered.
func New(rt *runtime.GatewayRuntime, logger *slog.Logger) *Server {
	r := chi.NewRouter()
	registerRoutes(r, rt, logger)

	// Wrap the entire router so cors intercepts OPTIONS before chi can respond.
	c := cors.New(cors.Options{
		AllowedOrigins:   rt.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	return &Server{router: c.Handler(r)}
}

// ServeHTTP implements http.Handler, allowing Server to be passed to httptest.NewServer.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Start begins listening on addr and blocks until the server returns an error.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
