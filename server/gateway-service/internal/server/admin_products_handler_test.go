package server_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/middleware"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"github.com/jsanca/go-folio/gateway-service/internal/server"
)

const testKID = "test-key-1"

// ── fixtures ──────────────────────────────────────────────────────────────────

var fixtureProjection = clients.CatalogProjection{
	Product: clients.CatalogProduct{
		ID:          1,
		ProductCode: "BAG-001",
		Title:       "Leather Tote Bag",
		Slug:        "leather-tote-bag",
		Active:      true,
	},
	Variants: []clients.CatalogVariant{
		{
			SKU:         "BAG-001-BRN-M",
			ColorName:   "Brown",
			RetailPrice: clients.CatalogMoney{AmountCents: 18900},
			Currency:    "USD",
			Active:      true,
		},
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

// fakeCatalogSrv starts an httptest server that returns fixtureProjection on any request.
func fakeCatalogSrv(t *testing.T) *httptest.Server {
	t.Helper()
	payload, _ := json.Marshal([]clients.CatalogProjection{fixtureProjection})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)
	return srv
}

// buildRuntime wires a GatewayRuntime suitable for tests.
//
// catalogURL must point to a running catalog httptest server.
// keycloakURL may be empty to use a permissive (no-auth) verifier.
// The inventory client points to an unreachable address; fetchStock falls back
// to OUT_OF_STOCK on every call, which is acceptable for these tests.
func buildRuntime(t *testing.T, catalogURL, keycloakURL string) *runtime.GatewayRuntime {
	t.Helper()
	inv, err := clients.NewInventoryClient("localhost:12345")
	if err != nil {
		t.Fatal("create test inventory client:", err)
	}
	t.Cleanup(func() { inv.Close() })

	auth, err := middleware.NewVerifier(context.Background(), keycloakURL, "folio")
	if err != nil {
		t.Fatal("create test verifier:", err)
	}
	return &runtime.GatewayRuntime{
		Catalog:           clients.NewCatalogClient(catalogURL),
		Inventory:         inv,
		Auth:              auth,
		LowStockThreshold: 5,
	}
}

// testOIDCSrv starts a minimal OIDC-compliant httptest server backed by a
// freshly generated RSA-2048 key pair.  It returns the server's base URL (pass
// as keycloakURL to buildRuntime) and a function that produces RS256-signed
// JWTs carrying the requested realm roles.
func testOIDCSrv(t *testing.T) (url string, signToken func(roles ...string) string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal("generate RSA key:", err)
	}

	// Allocate port before building handler so issuer URL is known.
	srv := httptest.NewUnstartedServer(nil)
	srvURL := "http://" + srv.Listener.Addr().String()
	issuer := srvURL + "/realms/folio"
	jwksURI := issuer + "/protocol/openid-connect/certs"

	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/realms/folio/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"issuer":   issuer,
				"jwks_uri": jwksURI,
				// Fields required by go-oidc provider discovery.
				"authorization_endpoint":                issuer + "/protocol/openid-connect/auth",
				"token_endpoint":                        issuer + "/protocol/openid-connect/token",
				"response_types_supported":              []string{"code"},
				"subject_types_supported":               []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/realms/folio/protocol/openid-connect/certs":
			w.Write(marshalJWKS(key.Public().(*rsa.PublicKey))) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	})
	srv.Start()
	t.Cleanup(srv.Close)

	signToken = func(roles ...string) string {
		return signRS256JWT(key, roles)
	}
	return srvURL, signToken
}

// marshalJWKS encodes the RSA public key as a JSON Web Key Set.
func marshalJWKS(pub *rsa.PublicKey) []byte {
	type JWK struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	eBytes := new(big.Int).SetInt64(int64(pub.E)).Bytes()
	data, _ := json.Marshal(struct {
		Keys []JWK `json:"keys"`
	}{
		Keys: []JWK{{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: testKID,
			N:   b64url(pub.N.Bytes()),
			E:   b64url(eBytes),
		}},
	})
	return data
}

// signRS256JWT produces a minimal RS256 JWT with the given realm roles.
func signRS256JWT(key *rsa.PrivateKey, roles []string) string {
	hdr, _ := json.Marshal(map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": testKID,
	})
	pay, _ := json.Marshal(map[string]any{
		"sub":          "test-user",
		"iat":          time.Now().Unix(),
		"exp":          time.Now().Add(time.Hour).Unix(),
		"realm_access": map[string]any{"roles": roles},
	})
	sigInput := b64url(hdr) + "." + b64url(pay)
	h := sha256.Sum256([]byte(sigInput))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	return sigInput + "." + b64url(sig)
}

// b64url encodes b as base64url with no padding.
func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// ── unit test ─────────────────────────────────────────────────────────────────

// TestAdminProductsHandler_ListProducts verifies that AdminProductsHandler
// calls catalog.ListProducts and shapes the response correctly.
// Inventory is unreachable, so all variants carry the OUT_OF_STOCK fallback.
func TestAdminProductsHandler_ListProducts(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	rt := buildRuntime(t, catalog.URL, "") // permissive — no JWT check
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := server.NewAdminProductsHandler(rt, logger)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body)
	}

	var products []struct {
		ID          int64  `json:"id"`
		ProductCode string `json:"productCode"`
		Title       string `json:"title"`
		Variants    []struct {
			SKU   string `json:"sku"`
			Stock struct {
				Available int32  `json:"available"`
				Status    string `json:"stockStatus"`
			} `json:"stock"`
		} `json:"variants"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&products); err != nil {
		t.Fatal("decode response:", err)
	}

	if len(products) != 1 {
		t.Fatalf("want 1 product, got %d", len(products))
	}
	p := products[0]
	if p.ProductCode != "BAG-001" {
		t.Errorf("productCode: want %q, got %q", "BAG-001", p.ProductCode)
	}
	if p.Title != "Leather Tote Bag" {
		t.Errorf("title: want %q, got %q", "Leather Tote Bag", p.Title)
	}
	if len(p.Variants) != 1 {
		t.Fatalf("want 1 variant, got %d", len(p.Variants))
	}
	v := p.Variants[0]
	if v.SKU != "BAG-001-BRN-M" {
		t.Errorf("sku: want %q, got %q", "BAG-001-BRN-M", v.SKU)
	}
	// inventory unreachable → handler falls back to available=0, OUT_OF_STOCK
	if v.Stock.Available != 0 {
		t.Errorf("available: want 0, got %d", v.Stock.Available)
	}
	if v.Stock.Status != "OUT_OF_STOCK" {
		t.Errorf("stockStatus: want %q, got %q", "OUT_OF_STOCK", v.Stock.Status)
	}
}

// ── integration tests ─────────────────────────────────────────────────────────

// TestAdminProductsHandler_AuthGates wires a full gateway httptest server and
// verifies that the auth middleware correctly gates /admin/products.
func TestAdminProductsHandler_AuthGates(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntime(t, catalog.URL, oidcURL)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "no token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-admin token",
			authHeader: "Bearer " + signToken("customer"),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin token",
			authHeader: "Bearer " + signToken("admin"),
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, gw.URL+"/admin/products", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}
