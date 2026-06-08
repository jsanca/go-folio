package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jsanca/go-folio/gateway-service/internal/server"
)

// TestCORS verifies that CORS preflight requests are handled correctly:
// allowed origins receive the ACAO header; disallowed origins do not.
func TestCORS(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	rt := buildRuntime(t, catalog.URL, "")
	rt.CORSOrigins = []string{"http://localhost:3000", "http://localhost:3001"}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	tests := []struct {
		name           string
		path           string
		origin         string
		wantCORSHeader bool
	}{
		{
			name:           "preflight /public/products from allowed origin",
			path:           "/public/products",
			origin:         "http://localhost:3000",
			wantCORSHeader: true,
		},
		{
			name:           "preflight /admin/products from allowed origin",
			path:           "/admin/products",
			origin:         "http://localhost:3001",
			wantCORSHeader: true,
		},
		{
			name:           "preflight from disallowed origin gets no CORS headers",
			path:           "/public/products",
			origin:         "http://evil.example.com",
			wantCORSHeader: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodOptions, gw.URL+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Access-Control-Request-Method", "GET")
			// The Fetch spec guarantees ACRH values are lowercase; rs/cors checks them
			// case-sensitively against its lowercased AllowedHeaders list.
			req.Header.Set("Access-Control-Request-Headers", "authorization")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			// rs/cors returns 204 No Content for preflights (correct per Fetch spec).
			if resp.StatusCode != http.StatusNoContent {
				t.Errorf("want 204, got %d", resp.StatusCode)
			}

			acao := resp.Header.Get("Access-Control-Allow-Origin")
			if tc.wantCORSHeader && acao == "" {
				t.Errorf("origin %q: want Access-Control-Allow-Origin to be set, got none", tc.origin)
			}
			if !tc.wantCORSHeader && acao != "" {
				t.Errorf("origin %q: want no Access-Control-Allow-Origin, got %q", tc.origin, acao)
			}
		})
	}
}
