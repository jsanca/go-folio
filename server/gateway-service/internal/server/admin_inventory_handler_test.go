package server_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
)

// ── fake gRPC inventory server ────────────────────────────────────────────────

// fakeInventoryServer is a configurable in-process gRPC server for tests.
type fakeInventoryServer struct {
	invpb.UnimplementedInventoryServiceServer
	listStockFn   func(*invpb.ListStockRequest) (*invpb.ListStockResponse, error)
	getStockFn    func(*invpb.GetStockRequest) (*invpb.GetStockResponse, error)
	adjustStockFn func(*invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error)
}

func (s *fakeInventoryServer) ListStock(_ context.Context, req *invpb.ListStockRequest) (*invpb.ListStockResponse, error) {
	if s.listStockFn != nil {
		return s.listStockFn(req)
	}
	return &invpb.ListStockResponse{}, nil
}

func (s *fakeInventoryServer) GetStock(_ context.Context, req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
	if s.getStockFn != nil {
		return s.getStockFn(req)
	}
	return nil, grpcstatus.Error(codes.NotFound, "not found")
}

func (s *fakeInventoryServer) AdjustStock(_ context.Context, req *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
	if s.adjustStockFn != nil {
		return s.adjustStockFn(req)
	}
	return nil, grpcstatus.Error(codes.NotFound, "not found")
}

// startFakeInventorySrv starts a real gRPC server with the fake implementation
// and returns its address. The server is shut down via t.Cleanup.
func startFakeInventorySrv(t *testing.T, fake *fakeInventoryServer) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("listen:", err)
	}
	srv := grpc.NewServer()
	invpb.RegisterInventoryServiceServer(srv, fake)
	go srv.Serve(lis) //nolint:errcheck
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

// buildRuntimeWithInv wires a GatewayRuntime using a live fake inventory gRPC server.
func buildRuntimeWithInv(t *testing.T, catalogURL, inventoryAddr, keycloakURL string) *runtime.GatewayRuntime {
	t.Helper()
	conn, err := grpc.NewClient(inventoryAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal("dial fake inventory:", err)
	}
	t.Cleanup(func() { conn.Close() })
	inv := &clients.InventoryClient{
		Svc: invpb.NewInventoryServiceClient(conn),
	}
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

// ── GET /admin/inventory ──────────────────────────────────────────────────────

// TestAdminInventory_ListStock verifies that GET /admin/inventory returns all
// SKUs with stock levels and the status field derived from LowStockThreshold.
// Auth is bypassed via the permissive verifier (empty keycloakURL).
func TestAdminInventory_ListStock(t *testing.T) {
	fake := &fakeInventoryServer{
		listStockFn: func(_ *invpb.ListStockRequest) (*invpb.ListStockResponse, error) {
			return &invpb.ListStockResponse{
				Items: []*invpb.StockRecord{
					{Sku: "BAG-001-BRN", Available: 20, Reserved: 0}, // IN_STOCK  (> threshold 5)
					{Sku: "BAG-001-BLK", Available: 3, Reserved: 1},  // LOW_STOCK (<= threshold 5)
					{Sku: "BAG-001-GRN", Available: 0, Reserved: 0},  // OUT_OF_STOCK
				},
			}, nil
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, "") // permissive — no JWT check
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/admin/inventory", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var items []struct {
		SKU       string `json:"sku"`
		Available int32  `json:"available"`
		Reserved  int32  `json:"reserved"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatal("decode:", err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}

	wantStatuses := map[string]string{
		"BAG-001-BRN": "IN_STOCK",
		"BAG-001-BLK": "LOW_STOCK",
		"BAG-001-GRN": "OUT_OF_STOCK",
	}
	for _, item := range items {
		if got, want := item.Status, wantStatuses[item.SKU]; got != want {
			t.Errorf("SKU %s: want status %q, got %q", item.SKU, want, got)
		}
	}
}

// ── GET /admin/inventory/{sku} ────────────────────────────────────────────────

// TestAdminInventory_GetStock_HappyPath verifies that a known SKU returns the
// correct stock fields and a derived status.
func TestAdminInventory_GetStock_HappyPath(t *testing.T) {
	fake := &fakeInventoryServer{
		getStockFn: func(req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
			if req.Sku == "WAL-001-BRN" {
				return &invpb.GetStockResponse{Sku: req.Sku, Available: 12, Reserved: 2}, nil
			}
			return nil, grpcstatus.Error(codes.NotFound, "not found")
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)
	token := "Bearer " + signToken("admin")

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/admin/inventory/WAL-001-BRN", nil)
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var item struct {
		SKU       string `json:"sku"`
		Available int32  `json:"available"`
		Reserved  int32  `json:"reserved"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		t.Fatal("decode:", err)
	}
	if item.SKU != "WAL-001-BRN" {
		t.Errorf("sku: want %q, got %q", "WAL-001-BRN", item.SKU)
	}
	if item.Available != 12 {
		t.Errorf("available: want 12, got %d", item.Available)
	}
	if item.Reserved != 2 {
		t.Errorf("reserved: want 2, got %d", item.Reserved)
	}
	if item.Status != "IN_STOCK" {
		t.Errorf("status: want %q, got %q", "IN_STOCK", item.Status)
	}
}

// TestAdminInventory_GetStock_NotFound verifies that an unknown SKU returns 404.
func TestAdminInventory_GetStock_NotFound(t *testing.T) {
	fake := &fakeInventoryServer{
		getStockFn: func(_ *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
			return nil, grpcstatus.Error(codes.NotFound, "not found")
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)
	token := "Bearer " + signToken("admin")

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/admin/inventory/GHOST-SKU", nil)
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

// ── PUT /admin/inventory/{sku} ────────────────────────────────────────────────

// TestAdminInventory_AdjustStock_HappyPath verifies that a valid delta is applied
// and the response carries the updated available count and derived status.
func TestAdminInventory_AdjustStock_HappyPath(t *testing.T) {
	fake := &fakeInventoryServer{
		adjustStockFn: func(req *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			return &invpb.AdjustStockResponse{Sku: req.Sku, Available: 8}, nil
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)
	token := "Bearer " + signToken("admin")

	body := strings.NewReader(`{"delta":-2,"reason":"sale"}`)
	req, _ := http.NewRequest(http.MethodPut, gw.URL+"/admin/inventory/BAG-001-BRN", body)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var result struct {
		SKU       string `json:"sku"`
		Available int32  `json:"available"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decode:", err)
	}
	if result.SKU != "BAG-001-BRN" {
		t.Errorf("sku: want %q, got %q", "BAG-001-BRN", result.SKU)
	}
	if result.Available != 8 {
		t.Errorf("available: want 8, got %d", result.Available)
	}
	if result.Status != "IN_STOCK" {
		t.Errorf("status: want %q, got %q", "IN_STOCK", result.Status)
	}
}

// TestAdminInventory_AdjustStock_NotFound verifies that an unknown SKU returns 404.
func TestAdminInventory_AdjustStock_NotFound(t *testing.T) {
	fake := &fakeInventoryServer{
		adjustStockFn: func(_ *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			return nil, grpcstatus.Error(codes.NotFound, "sku not found")
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)
	token := "Bearer " + signToken("admin")

	body := strings.NewReader(`{"delta":1,"reason":"restock"}`)
	req, _ := http.NewRequest(http.MethodPut, gw.URL+"/admin/inventory/GHOST-SKU", body)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

// TestAdminInventory_AdjustStock_InsufficientStock verifies that a delta that
// would drive available below zero returns 422.
func TestAdminInventory_AdjustStock_InsufficientStock(t *testing.T) {
	fake := &fakeInventoryServer{
		adjustStockFn: func(_ *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
			return nil, grpcstatus.Error(codes.FailedPrecondition, "insufficient stock")
		},
	}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)
	token := "Bearer " + signToken("admin")

	body := strings.NewReader(`{"delta":-999,"reason":"sale"}`)
	req, _ := http.NewRequest(http.MethodPut, gw.URL+"/admin/inventory/BAG-001-BRN", body)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", resp.StatusCode)
	}
}

// ── auth gates ────────────────────────────────────────────────────────────────

// TestAdminInventory_AuthGates verifies that all three inventory endpoints
// return 401 without a token and 403 with a non-admin token.
func TestAdminInventory_AuthGates(t *testing.T) {
	fake := &fakeInventoryServer{}
	addr := startFakeInventorySrv(t, fake)
	catalog := fakeCatalogSrv(t)
	oidcURL, signToken := testOIDCSrv(t)
	rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/admin/inventory", ""},
		{http.MethodGet, "/admin/inventory/BAG-001", ""},
		{http.MethodPut, "/admin/inventory/BAG-001", `{"delta":1,"reason":"test"}`},
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
				var bodyReader *strings.Reader
				if ep.body != "" {
					bodyReader = strings.NewReader(ep.body)
				}
				var req *http.Request
				if bodyReader != nil {
					req, _ = http.NewRequest(ep.method, gw.URL+ep.path, bodyReader)
					req.Header.Set("Content-Type", "application/json")
				} else {
					req, _ = http.NewRequest(ep.method, gw.URL+ep.path, nil)
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

// ── SSE publish after adjustStock ─────────────────────────────────────────────

// adjustAndCapture performs a PUT /admin/inventory/{sku} and returns the first
// StockEvent received by a broker subscriber, or fails the test on timeout.
func adjustAndCapture(t *testing.T, gw *httptest.Server, token, sku, body string, rt *runtime.GatewayRuntime) sse.StockEvent {
	t.Helper()
	ch := rt.Events.Subscribe()
	t.Cleanup(func() { rt.Events.Unsubscribe(ch) })

	req, _ := http.NewRequest(http.MethodPut, gw.URL+"/admin/inventory/"+sku, strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("adjust: want 200, got %d", resp.StatusCode)
	}

	select {
	case event := <-ch:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE event")
		return sse.StockEvent{}
	}
}

// TestAdjustStock_PublishesEvent verifies that a successful adjustment publishes
// a StockEvent with the correct SKU, available count, and eventType.
func TestAdjustStock_PublishesEvent(t *testing.T) {
	tests := []struct {
		name          string
		available     int32
		wantEventType string
	}{
		{"stock.updated when above threshold", 10, "stock.updated"},
		{"stock.low when at threshold", 5, "stock.low"},
		{"stock.low when below threshold", 3, "stock.low"},
		{"stock.out when zero", 0, "stock.out"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeInventoryServer{
				adjustStockFn: func(req *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
					return &invpb.AdjustStockResponse{Sku: req.Sku, Available: tc.available}, nil
				},
			}
			addr := startFakeInventorySrv(t, fake)
			catalog := fakeCatalogSrv(t)
			oidcURL, signToken := testOIDCSrv(t)
			rt := buildRuntimeWithInv(t, catalog.URL, addr, oidcURL)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			gw := httptest.NewServer(server.New(rt, logger))
			t.Cleanup(gw.Close)

			event := adjustAndCapture(t, gw, "Bearer "+signToken("admin"), "BAG-001", `{"delta":1,"reason":"test"}`, rt)

			if event.SKU != "BAG-001" {
				t.Errorf("SKU: want %q, got %q", "BAG-001", event.SKU)
			}
			if event.Available != tc.available {
				t.Errorf("Available: want %d, got %d", tc.available, event.Available)
			}
			if event.EventType != tc.wantEventType {
				t.Errorf("EventType: want %q, got %q", tc.wantEventType, event.EventType)
			}
			if event.OccurredAt.IsZero() {
				t.Error("OccurredAt must not be zero")
			}
		})
	}
}
