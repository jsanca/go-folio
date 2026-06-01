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
	"strings"
	"testing"
	"time"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/middleware"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"github.com/jsanca/go-folio/gateway-service/internal/server"
	"github.com/jsanca/go-folio/gateway-service/internal/sse"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
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
		Events:            sse.NewBroker(),
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

// ── mutation proxy tests ──────────────────────────────────────────────────────

// TestAdminProductsMutations_Proxy verifies that POST, PATCH, and DELETE under
// /admin/products proxy through to catalog-service and return its status code.
func TestAdminProductsMutations_Proxy(t *testing.T) {
	oidcURL, signToken := testOIDCSrv(t)
	adminToken := "Bearer " + signToken("admin")

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		catalogStatus int
		wantStatus    int
	}{
		{
			name:          "POST creates product → 201",
			method:        http.MethodPost,
			path:          "/admin/products",
			body:          `{"productCode":"BAG-002","title":"Canvas Bag","slug":"canvas-bag"}`,
			catalogStatus: http.StatusCreated,
			wantStatus:    http.StatusCreated,
		},
		{
			name:          "POST missing fields → 400 from catalog",
			method:        http.MethodPost,
			path:          "/admin/products",
			body:          `{}`,
			catalogStatus: http.StatusBadRequest,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "PATCH updates product → 200",
			method:        http.MethodPatch,
			path:          "/admin/products/1",
			body:          `{"title":"Updated Title"}`,
			catalogStatus: http.StatusOK,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "DELETE removes product → 204",
			method:        http.MethodDelete,
			path:          "/admin/products/1",
			body:          "",
			catalogStatus: http.StatusNoContent,
			wantStatus:    http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.catalogStatus)
			}))
			t.Cleanup(catalog.Close)

			rt := buildRuntime(t, catalog.URL, oidcURL)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			gw := httptest.NewServer(server.New(rt, logger))
			t.Cleanup(gw.Close)

			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}
			req, err := http.NewRequest(tc.method, gw.URL+tc.path, bodyReader)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", adminToken)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
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

// TestAdminProductsMutations_AuthGates verifies that all three mutation endpoints
// return 401 without a token and 403 with a non-admin token.
func TestAdminProductsMutations_AuthGates(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntime(t, catalog.URL, oidcURL)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/admin/products"},
		{http.MethodPatch, "/admin/products/1"},
		{http.MethodDelete, "/admin/products/1"},
	}

	authCases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"non-admin token", "Bearer " + signToken("customer"), http.StatusForbidden},
	}

	for _, ep := range endpoints {
		for _, ac := range authCases {
			t.Run(ep.method+" "+ep.path+"/"+ac.name, func(t *testing.T) {
				req, err := http.NewRequest(ep.method, gw.URL+ep.path, nil)
				if err != nil {
					t.Fatal(err)
				}
				if ac.header != "" {
					req.Header.Set("Authorization", ac.header)
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				if resp.StatusCode != ac.wantStatus {
					t.Errorf("want %d, got %d", ac.wantStatus, resp.StatusCode)
				}
			})
		}
	}
}

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

// TestAdminProducts_FullFields verifies that GET /admin/products returns the
// full product shape including shortDescription, department, category, active,
// and that stock is hydrated from inventory-service (not the fallback values).
func TestAdminProducts_FullFields(t *testing.T) {
	fake := &fakeInventoryServer{
		getStockFn: func(req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
			return &invpb.GetStockResponse{Sku: req.GetSku(), Available: 20, Reserved: 2}, nil
		},
	}
	invAddr := startFakeInventorySrv(t, fake)

	fullFixture := clients.CatalogProjection{
		Product: clients.CatalogProduct{
			ID:               2,
			ProductCode:      "WAL-001",
			Title:            "Slim Wallet",
			Slug:             "slim-wallet",
			ShortDescription: "A slim leather wallet",
			Department:       "Wallets",
			Category:         "Slim",
			Active:           true,
		},
		Variants: []clients.CatalogVariant{
			{SKU: "WAL-001-BRN", Currency: "USD", Active: true},
		},
	}

	payload, _ := json.Marshal([]clients.CatalogProjection{fullFixture})
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload) //nolint:errcheck
	}))
	t.Cleanup(catalog.Close)

	rt := buildRuntimeWithInv(t, catalog.URL, invAddr, "") // permissive — no JWT check
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := server.NewAdminProductsHandler(rt, logger)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body)
	}

	var products []struct {
		ID               int64  `json:"id"`
		ProductCode      string `json:"productCode"`
		Title            string `json:"title"`
		Slug             string `json:"slug"`
		ShortDescription string `json:"shortDescription"`
		Department       string `json:"department"`
		Category         string `json:"category"`
		Active           bool   `json:"active"`
		Variants         []struct {
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
	if p.ShortDescription != "A slim leather wallet" {
		t.Errorf("shortDescription: want %q, got %q", "A slim leather wallet", p.ShortDescription)
	}
	if p.Department != "Wallets" {
		t.Errorf("department: want %q, got %q", "Wallets", p.Department)
	}
	if p.Category != "Slim" {
		t.Errorf("category: want %q, got %q", "Slim", p.Category)
	}
	if !p.Active {
		t.Errorf("active: want true, got false")
	}

	// Verify stock is hydrated from inventory-service, not the OUT_OF_STOCK fallback.
	if len(p.Variants) != 1 {
		t.Fatalf("want 1 variant, got %d", len(p.Variants))
	}
	v := p.Variants[0]
	if v.SKU != "WAL-001-BRN" {
		t.Errorf("sku: want %q, got %q", "WAL-001-BRN", v.SKU)
	}
	if v.Stock.Available <= 0 {
		t.Errorf("stock.available: want > 0, got %d", v.Stock.Available)
	}
	if v.Stock.Status == "OUT_OF_STOCK" {
		t.Errorf("stock.stockStatus: want IN_STOCK or LOW_STOCK, got OUT_OF_STOCK")
	}
}

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

// ── POST /admin/products/{id}/variants saga tests ─────────────────────────────

// sagaCatalogSrv builds a configurable catalog httptest.Server for saga tests.
// postFn handles POST /catalog/products/*/variants.
// deleteFn handles DELETE /catalog/products/*/variants/*.
func sagaCatalogSrv(t *testing.T,
	postFn func(w http.ResponseWriter, r *http.Request),
	deleteFn func(w http.ResponseWriter, r *http.Request),
) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isVariants := strings.Contains(r.URL.Path, "/variants")
		switch {
		case r.Method == http.MethodPost && isVariants && postFn != nil:
			postFn(w, r)
		case r.Method == http.MethodDelete && isVariants && deleteFn != nil:
			deleteFn(w, r)
		default:
			// default catalog behaviour (product list etc.)
			w.Header().Set("Content-Type", "application/json")
			payload, _ := json.Marshal([]clients.CatalogProjection{fixtureProjection})
			w.Write(payload) //nolint:errcheck
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// variantFixtureJSON is what catalog returns after a successful variant creation.
const variantFixtureJSON = `{"sku":"BAG-001-RED","colorName":"Red","currency":"USD","active":true}`

// TestAddVariantSaga_HappyPath verifies that POST /admin/products/{id}/variants
// calls both catalog and inventory and returns 201 on success.
func TestAddVariantSaga_HappyPath(t *testing.T) {
	var inventorySeeded bool
	fake := &fakeInventoryServer{
		adjustStockFn: func(req *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			if req.Sku == "BAG-001-RED" && req.Delta == 0 {
				inventorySeeded = true
			}
			return &invpb.AdjustStockResponse{Sku: req.Sku, Available: 0}, nil
		},
	}
	invAddr := startFakeInventorySrv(t, fake)

	var catalogCalled bool
	catalog := sagaCatalogSrv(t,
		func(w http.ResponseWriter, r *http.Request) {
			catalogCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, variantFixtureJSON) //nolint:errcheck
		},
		nil,
	)

	rt := buildRuntimeWithInv(t, catalog.URL, invAddr, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	body := strings.NewReader(`{"sku":"BAG-001-RED","retailPriceCents":1000,"currency":"USD"}`)
	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/admin/products/1/variants", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("want 201, got %d", resp.StatusCode)
	}
	if !catalogCalled {
		t.Error("catalog POST /variants was not called")
	}
	if !inventorySeeded {
		t.Error("inventory SeedSKU was not called with correct args")
	}
}

// TestAddVariantSaga_InventoryFails verifies that when inventory seeding fails
// the saga calls catalog DELETE (compensation) and returns 503.
func TestAddVariantSaga_InventoryFails(t *testing.T) {
	fake := &fakeInventoryServer{
		adjustStockFn: func(_ *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			return nil, grpcstatus.Error(codes.Internal, "inventory unavailable")
		},
	}
	invAddr := startFakeInventorySrv(t, fake)

	var compensationCalled bool
	catalog := sagaCatalogSrv(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, variantFixtureJSON) //nolint:errcheck
		},
		func(w http.ResponseWriter, r *http.Request) {
			compensationCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
	)

	rt := buildRuntimeWithInv(t, catalog.URL, invAddr, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	body := strings.NewReader(`{"sku":"BAG-001-RED","retailPriceCents":1000,"currency":"USD"}`)
	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/admin/products/1/variants", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", resp.StatusCode)
	}
	if !compensationCalled {
		t.Error("catalog DELETE /variants was not called as compensation")
	}
}

// TestAddVariantSaga_CompensationFails verifies that when both inventory seeding
// and catalog compensation fail the handler still returns 503.
func TestAddVariantSaga_CompensationFails(t *testing.T) {
	fake := &fakeInventoryServer{
		adjustStockFn: func(_ *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			return nil, grpcstatus.Error(codes.Internal, "inventory unavailable")
		},
	}
	invAddr := startFakeInventorySrv(t, fake)

	catalog := sagaCatalogSrv(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, variantFixtureJSON) //nolint:errcheck
		},
		func(w http.ResponseWriter, r *http.Request) {
			// compensation itself fails
			w.WriteHeader(http.StatusInternalServerError)
		},
	)

	rt := buildRuntimeWithInv(t, catalog.URL, invAddr, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	body := strings.NewReader(`{"sku":"BAG-001-RED","retailPriceCents":1000,"currency":"USD"}`)
	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/admin/products/1/variants", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", resp.StatusCode)
	}
}
