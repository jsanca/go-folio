// Package clients contains downstream service clients used by the gateway.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrNotFound is returned when catalog-service responds with 404.
var ErrNotFound = errors.New("catalog: not found")

// CatalogMoney mirrors catalog-service's Money JSON shape.
type CatalogMoney struct {
	AmountCents int64 `json:"amountCents"`
}

// CatalogVariant is the subset of catalog-service's ProductVariant used by the gateway.
type CatalogVariant struct {
	SKU             string        `json:"sku"`
	ColorName       string        `json:"colorName"`
	ColorSlug       string        `json:"colorSlug"`
	PrimaryColorHex string        `json:"primaryColorHex"`
	RetailPrice     CatalogMoney  `json:"retailPrice"`
	SalePrice       *CatalogMoney `json:"salePrice"`
	Currency        string        `json:"currency"`
	Active          bool          `json:"active"`
}

// CatalogProduct is the subset of catalog-service's Product used by the gateway.
type CatalogProduct struct {
	ID               int64  `json:"id"`
	ProductCode      string `json:"productCode"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	ShortDescription string `json:"shortDescription"`
	Department       string `json:"department"`
	Category         string `json:"category"`
	Active           bool   `json:"active"`
}

// CatalogProjection mirrors catalog-service's ProductProjection JSON shape.
type CatalogProjection struct {
	Product  CatalogProduct   `json:"product"`
	Variants []CatalogVariant `json:"variants"`
}

// CatalogClient is an HTTP client for the catalog-service REST API.
type CatalogClient struct {
	baseURL string
	http    *http.Client
}

// NewCatalogClient creates a CatalogClient targeting baseURL.
func NewCatalogClient(baseURL string) *CatalogClient {
	return &CatalogClient{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

// ListProducts fetches all products with their variants from catalog-service.
func (c *CatalogClient) ListProducts(ctx context.Context) ([]CatalogProjection, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/products", nil)
	if err != nil {
		return nil, fmt.Errorf("build list products request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list products: unexpected status %d", resp.StatusCode)
	}
	var result []CatalogProjection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode list products response: %w", err)
	}
	return result, nil
}

// GetVariantBySKU fetches the product projection for a single variant SKU.
// Returns ErrNotFound if catalog-service responds with 404.
func (c *CatalogClient) GetVariantBySKU(ctx context.Context, sku string) (*CatalogProjection, error) {
	u := c.baseURL + "/catalog/variants/" + url.PathEscape(sku)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build get variant request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get variant: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get variant: unexpected status %d", resp.StatusCode)
	}
	var result CatalogProjection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode get variant response: %w", err)
	}
	return &result, nil
}

// ProxyRequest forwards a mutating request to the catalog-service and returns
// the upstream status code and raw response body.
func (c *CatalogClient) ProxyRequest(ctx context.Context, method, path string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("build proxy request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("proxy request: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read proxy response: %w", err)
	}
	return resp.StatusCode, b, nil
}

// Close is a no-op; http.Client holds no persistent connection to release.
// Implements io.Closer for CompositeRuntime compatibility.
func (c *CatalogClient) Close() error {
	return nil
}
