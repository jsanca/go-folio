package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/server"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// ── fixtures ──────────────────────────────────────────────────────────────────

var publicFixtureProjections = []clients.CatalogProjection{
	{
		Product: clients.CatalogProduct{
			ID:          1,
			ProductCode: "BAG-001",
			Title:       "Leather Tote",
			Slug:        "leather-tote",
		},
		Variants: []clients.CatalogVariant{
			{SKU: "BAG-001-BRN", ColorName: "Brown", RetailPrice: clients.CatalogMoney{AmountCents: 18900}, Currency: "USD", Active: true},
			{SKU: "BAG-001-BLK", ColorName: "Black", RetailPrice: clients.CatalogMoney{AmountCents: 18900}, Currency: "USD", Active: true},
		},
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

// publicCatalogSrv starts a fake catalog HTTP server that handles both endpoints
// used by ProductsHandler:
//
//	GET /products                → JSON array of all projections
//	GET /catalog/variants/{sku} → matching CatalogProjection, or 404
func publicCatalogSrv(t *testing.T, projections []clients.CatalogProjection) *httptest.Server {
	t.Helper()
	listPayload, _ := json.Marshal(projections)

	type entry struct {
		proj    clients.CatalogProjection
		variant clients.CatalogVariant
	}
	bysku := make(map[string]entry)
	for _, p := range projections {
		for _, v := range p.Variants {
			bysku[v.SKU] = entry{proj: p, variant: v}
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/products":
			w.Write(listPayload) //nolint:errcheck
		case strings.HasPrefix(r.URL.Path, "/catalog/variants/"):
			sku := strings.TrimPrefix(r.URL.Path, "/catalog/variants/")
			e, ok := bysku[sku]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"not found"}}`)) //nolint:errcheck
				return
			}
			payload, _ := json.Marshal(clients.CatalogProjection{
				Product:  e.proj.Product,
				Variants: []clients.CatalogVariant{e.variant},
			})
			w.Write(payload) //nolint:errcheck
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ── GET /products ─────────────────────────────────────────────────────────────

// TestPublicProducts_ListProducts_HappyPath verifies that GET /products returns
// all products with variants and live stock hydrated from inventory.
func TestPublicProducts_ListProducts_HappyPath(t *testing.T) {
	fake := &fakeInventoryServer{
		getStockFn: func(req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
			switch req.Sku {
			case "BAG-001-BRN":
				return &invpb.GetStockResponse{Sku: req.Sku, Available: 10, Reserved: 1}, nil
			case "BAG-001-BLK":
				return &invpb.GetStockResponse{Sku: req.Sku, Available: 3, Reserved: 0}, nil
			}
			return nil, grpcstatus.Error(codes.NotFound, "not found")
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := publicCatalogSrv(t, publicFixtureProjections)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, "") // permissive — no JWT check
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	resp, err := http.Get(gw.URL + "/products")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var products []struct {
		ProductCode string `json:"productCode"`
		Variants    []struct {
			SKU   string `json:"sku"`
			Stock struct {
				Available int32  `json:"available"`
				Reserved  int32  `json:"reserved"`
				Status    string `json:"stockStatus"`
			} `json:"stock"`
		} `json:"variants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		t.Fatal("decode:", err)
	}
	if len(products) != 1 {
		t.Fatalf("want 1 product, got %d", len(products))
	}
	if products[0].ProductCode != "BAG-001" {
		t.Errorf("productCode: want BAG-001, got %s", products[0].ProductCode)
	}
	if len(products[0].Variants) != 2 {
		t.Fatalf("want 2 variants, got %d", len(products[0].Variants))
	}

	type stockSummary struct{ available, reserved int32; status string }
	bysku := map[string]stockSummary{}
	for _, v := range products[0].Variants {
		bysku[v.SKU] = stockSummary{v.Stock.Available, v.Stock.Reserved, v.Stock.Status}
	}

	if s := bysku["BAG-001-BRN"]; s.available != 10 || s.reserved != 1 || s.status != "IN_STOCK" {
		t.Errorf("BAG-001-BRN: want available=10 reserved=1 IN_STOCK, got %+v", s)
	}
	if s := bysku["BAG-001-BLK"]; s.available != 3 || s.reserved != 0 || s.status != "LOW_STOCK" {
		t.Errorf("BAG-001-BLK: want available=3 reserved=0 LOW_STOCK, got %+v", s)
	}
}

// TestPublicProducts_ListProducts_InventoryUnreachable verifies that GET /products
// still returns 200 with all variants, falling back to available=0 / OUT_OF_STOCK
// when inventory-service is unreachable. buildRuntime hardwires localhost:12345.
func TestPublicProducts_ListProducts_InventoryUnreachable(t *testing.T) {
	catalog := publicCatalogSrv(t, publicFixtureProjections)
	rt := buildRuntime(t, catalog.URL, "") // inventory unreachable at localhost:12345
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	resp, err := http.Get(gw.URL + "/products")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var products []struct {
		Variants []struct {
			Stock struct {
				Available int32  `json:"available"`
				Status    string `json:"stockStatus"`
			} `json:"stock"`
		} `json:"variants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		t.Fatal("decode:", err)
	}
	if len(products) == 0 {
		t.Fatal("want at least one product")
	}
	for _, p := range products {
		for _, v := range p.Variants {
			if v.Stock.Available != 0 {
				t.Errorf("want available=0 on unreachable inventory, got %d", v.Stock.Available)
			}
			if v.Stock.Status != "OUT_OF_STOCK" {
				t.Errorf("want OUT_OF_STOCK on unreachable inventory, got %s", v.Stock.Status)
			}
		}
	}
}

// ── GET /products/{sku} ───────────────────────────────────────────────────────

// TestPublicProducts_GetBySKU_HappyPath verifies that GET /products/{sku}
// returns the single-variant product shape with live stock hydrated.
func TestPublicProducts_GetBySKU_HappyPath(t *testing.T) {
	fake := &fakeInventoryServer{
		getStockFn: func(req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
			return &invpb.GetStockResponse{Sku: req.Sku, Available: 7, Reserved: 1}, nil
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := publicCatalogSrv(t, publicFixtureProjections)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	resp, err := http.Get(gw.URL + "/products/BAG-001-BRN")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var product struct {
		ProductCode string `json:"productCode"`
		SKU         string `json:"sku"`
		ColorName   string `json:"colorName"`
		Stock       struct {
			Available int32  `json:"available"`
			Reserved  int32  `json:"reserved"`
			Status    string `json:"stockStatus"`
		} `json:"stock"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		t.Fatal("decode:", err)
	}
	if product.ProductCode != "BAG-001" {
		t.Errorf("productCode: want BAG-001, got %s", product.ProductCode)
	}
	if product.SKU != "BAG-001-BRN" {
		t.Errorf("sku: want BAG-001-BRN, got %s", product.SKU)
	}
	if product.ColorName != "Brown" {
		t.Errorf("colorName: want Brown, got %s", product.ColorName)
	}
	if product.Stock.Available != 7 {
		t.Errorf("available: want 7, got %d", product.Stock.Available)
	}
	if product.Stock.Reserved != 1 {
		t.Errorf("reserved: want 1, got %d", product.Stock.Reserved)
	}
	if product.Stock.Status != "IN_STOCK" {
		t.Errorf("status: want IN_STOCK, got %s", product.Stock.Status)
	}
}

// TestPublicProducts_GetBySKU_NotFound verifies that a SKU absent from
// catalog-service causes the gateway to return 404.
func TestPublicProducts_GetBySKU_NotFound(t *testing.T) {
	fake := &fakeInventoryServer{}
	addr := startFakeInventorySrv(t, fake)
	catalog := publicCatalogSrv(t, publicFixtureProjections) // GHOST-SKU not in fixture
	rt := buildRuntimeWithInv(t, catalog.URL, addr, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	resp, err := http.Get(gw.URL + "/products/GHOST-SKU-999")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

// ── deriveStockStatus ─────────────────────────────────────────────────────────

// TestPublicProducts_DeriveStockStatus covers all boundary cases of the stock
// status derivation logic. Each case is exercised end-to-end through
// GET /products with a fake inventory returning the specified available count.
// Checking the stockStatus field in the response is equivalent to a direct unit
// test since deriveStockStatus is the only thing that determines that field.
func TestPublicProducts_DeriveStockStatus(t *testing.T) {
	tests := []struct {
		name       string
		available  int32
		threshold  int
		wantStatus string
	}{
		{"available=0 → OUT_OF_STOCK",                     0, 5, "OUT_OF_STOCK"},
		{"available==threshold → LOW_STOCK",               5, 5, "LOW_STOCK"},
		{"available<threshold → LOW_STOCK",                3, 5, "LOW_STOCK"},
		{"available>threshold → IN_STOCK",                 6, 5, "IN_STOCK"},
		{"threshold=0, available=1 → IN_STOCK",            1, 0, "IN_STOCK"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeInventoryServer{
				getStockFn: func(_ *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
					return &invpb.GetStockResponse{Sku: "SKU-TEST", Available: tc.available}, nil
				},
			}
			addr := startFakeInventorySrv(t, fake)

			oneVariant := []clients.CatalogProjection{{
				Product:  clients.CatalogProduct{ID: 1, ProductCode: "X", Title: "X", Slug: "x"},
				Variants: []clients.CatalogVariant{{SKU: "SKU-TEST", Currency: "USD"}},
			}}
			catalog := publicCatalogSrv(t, oneVariant)

			rt := buildRuntimeWithInv(t, catalog.URL, addr, "")
			rt.LowStockThreshold = tc.threshold

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			gw := httptest.NewServer(server.New(rt, logger))
			t.Cleanup(gw.Close)

			resp, err := http.Get(gw.URL + "/products")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			var products []struct {
				Variants []struct {
					Stock struct {
						Status string `json:"stockStatus"`
					} `json:"stock"`
				} `json:"variants"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
				t.Fatal("decode:", err)
			}
			if len(products) != 1 || len(products[0].Variants) != 1 {
				t.Fatalf("unexpected response shape: %d products", len(products))
			}
			if got := products[0].Variants[0].Stock.Status; got != tc.wantStatus {
				t.Errorf("available=%d threshold=%d: want %s, got %s",
					tc.available, tc.threshold, tc.wantStatus, got)
			}
		})
	}
}
