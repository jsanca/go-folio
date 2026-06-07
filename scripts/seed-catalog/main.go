// seed-catalog uploads product data from a Silver export directory to the
// go-folio gateway, with images stored in MinIO.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ── Silver JSON schema ────────────────────────────────────────────────────────

// silverProduct is the shape of a single *.json file in the silver-dir.
type silverProduct struct {
	SKU              string          `json:"sku"`
	Title            string          `json:"title"`
	Slug             string          `json:"slug"`
	ShortDescription string          `json:"shortDescription"`
	Department       string          `json:"department"`
	Category         string          `json:"category"`
	Active           bool            `json:"active"`
	Variants         []silverVariant `json:"variants"`
	Images           []silverImage   `json:"images"`
}

type silverVariant struct {
	SKU             string `json:"sku"`
	ColorName       string `json:"colorName"`
	ColorSlug       string `json:"colorSlug"`
	PrimaryColorHex string `json:"primaryColorHex"`
	RetailPrice     int64  `json:"retailPriceCents"`
	Currency        string `json:"currency"`
	Active          bool   `json:"active"`
}

type silverImage struct {
	File    string `json:"file"`    // relative to --images-dir
	AltText string `json:"altText"` // optional
}

// ── Gateway API payloads ──────────────────────────────────────────────────────

type createProductPayload struct {
	ProductCode      string `json:"productCode"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	ShortDescription string `json:"shortDescription"`
	Department       string `json:"department"`
	Category         string `json:"category"`
	Active           bool   `json:"active"`
}

type createdProduct struct {
	ID int64 `json:"id"`
}

type createVariantPayload struct {
	SKU             string `json:"sku"`
	ColorName       string `json:"colorName"`
	ColorSlug       string `json:"colorSlug"`
	PrimaryColorHex string `json:"primaryColorHex"`
	RetailPrice     int64  `json:"retailPriceCents"`
	Currency        string `json:"currency"`
	Active          bool   `json:"active"`
}

// ── Keycloak token ────────────────────────────────────────────────────────────

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func fetchToken(keycloakURL, realm, username, password string) (*tokenResponse, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakURL, realm)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "gateway")
	form.Set("username", username)
	form.Set("password", password)

	resp, err := http.PostForm(endpoint, form)
	if err != nil {
		return nil, fmt.Errorf("keycloak token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("keycloak returned %d: %s", resp.StatusCode, body)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	return &tok, nil
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

type client struct {
	gateway      string
	keycloakURL  string
	realm        string
	username     string
	password     string
	accessToken  string
	tokenExpires time.Time
	http         *http.Client
}

func newClient(gateway, keycloakURL, realm, username, password string) *client {
	return &client{
		gateway:     gateway,
		keycloakURL: keycloakURL,
		realm:       realm,
		username:    username,
		password:    password,
		http:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *client) ensureToken() error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpires) {
		return nil
	}
	tok, err := fetchToken(c.keycloakURL, c.realm, c.username, c.password)
	if err != nil {
		return err
	}
	c.accessToken = tok.AccessToken
	// Refresh 30 s before actual expiry.
	c.tokenExpires = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	return nil
}

// doJSON executes a JSON request and returns the HTTP status, raw response body, and any error.
// On a 401 response it refreshes the token and retries exactly once.
func (c *client) doJSON(method, path string, body any, out any) (int, []byte, error) {
	return c.doJSONWithRetry(method, path, body, out, false)
}

func (c *client) doJSONWithRetry(method, path string, body any, out any, retried bool) (int, []byte, error) {
	if err := c.ensureToken(); err != nil {
		return 0, nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.gateway+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if retried {
			return 0, nil, fmt.Errorf("authentication failed after token refresh")
		}
		c.accessToken = ""
		return c.doJSONWithRetry(method, path, body, out, true)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp.StatusCode, respBody, fmt.Errorf("decode response (%d): %w — body: %s", resp.StatusCode, err, respBody)
		}
	}
	return resp.StatusCode, respBody, nil
}

// ── MinIO helpers ─────────────────────────────────────────────────────────────

const bucket = "folio"

func initMinio(endpoint, user, pass string) (*minio.Client, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(user, pass, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket check: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
		slog.Info("created bucket", "name", bucket)
	}
	return mc, nil
}

func uploadImage(mc *minio.Client, slug, fullPath string) (string, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("open image %s: %w", fullPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectName := "products/" + slug + "/" + filepath.Base(fullPath)
	_, err = mc.PutObject(context.Background(), bucket, objectName, f, info.Size(),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", fullPath, err)
	}
	return objectName, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	var (
		gateway   = flag.String("gateway", "http://localhost:8090", "gateway base URL")
		keycloak  = flag.String("keycloak", "http://localhost:8080", "Keycloak base URL")
		realm     = flag.String("realm", "folio", "Keycloak realm")
		user      = flag.String("user", "admin@folio.dev", "admin username")
		password  = flag.String("password", "admin123", "admin password")
		silverDir = flag.String("silver-dir", "./silver", "directory of Silver *.json files")
		imagesDir = flag.String("images-dir", "./images", "directory of product images")
		minioURL  = flag.String("minio-url", "localhost:9000", "MinIO endpoint (host:port)")
		minioUser = flag.String("minio-user", "folio", "MinIO root user")
		minioPass = flag.String("minio-pass", "folio1234", "MinIO root password")
		dryRun    = flag.Bool("dry-run", false, "parse and log actions without calling APIs")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	pattern := filepath.Join(*silverDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		slog.Error("no Silver JSON files found", "pattern", pattern)
		os.Exit(1)
	}

	var mc *minio.Client
	if !*dryRun {
		mc, err = initMinio(*minioURL, *minioUser, *minioPass)
		if err != nil {
			slog.Error("MinIO init failed", "err", err)
			os.Exit(1)
		}
		slog.Info("MinIO ready", "bucket", bucket)
	}

	cli := newClient(*gateway, *keycloak, *realm, *user, *password)

	var seeded, skipped, failed int

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			slog.Error("read file", "file", f, "err", err)
			failed++
			continue
		}

		var sp silverProduct
		if err := json.Unmarshal(data, &sp); err != nil {
			slog.Error("parse Silver JSON", "file", f, "err", err)
			failed++
			continue
		}

		productCode := sp.SKU
		if productCode == "" {
			productCode = sp.Slug
		}

		log := slog.With("product", productCode, "title", sp.Title)

		if *dryRun {
			log.Info("dry-run: would create product",
				"variants", len(sp.Variants),
				"images", len(sp.Images),
			)
			seeded++
			continue
		}

		// Create product.
		var created createdProduct
		status, _, err := cli.doJSON(http.MethodPost, "/admin/products",
			createProductPayload{
				ProductCode:      productCode,
				Title:            sp.Title,
				Slug:             sp.Slug,
				ShortDescription: sp.ShortDescription,
				Department:       sp.Department,
				Category:         sp.Category,
				Active:           sp.Active,
			}, &created)
		if err != nil {
			log.Error("create product", "err", err)
			failed++
			continue
		}
		if status == http.StatusConflict {
			log.Info("product already exists, skipping", "status", status)
			skipped++
			continue
		}
		if status != http.StatusCreated {
			log.Error("create product unexpected status", "status", status)
			failed++
			continue
		}
		log.Info("product created", "id", created.ID)

		// Add variants.
		for _, v := range sp.Variants {
			price := v.RetailPrice
			vlog := log.With("sku", v.SKU)
			vstatus, _, err := cli.doJSON(http.MethodPost,
				fmt.Sprintf("/admin/products/%d/variants", created.ID),
				createVariantPayload{
					SKU:             v.SKU,
					ColorName:       v.ColorName,
					ColorSlug:       v.ColorSlug,
					PrimaryColorHex: v.PrimaryColorHex,
					RetailPrice:     price,
					Currency:        v.Currency,
					Active:          v.Active,
				}, nil)
			if err != nil || (vstatus != http.StatusCreated && vstatus != http.StatusConflict) {
				vlog.Error("add variant", "status", vstatus, "err", err)
			} else {
				vlog.Info("variant seeded", "status", vstatus)
			}
		}

		// Upload images from {images-dir}/{slug}/ and track the first success.
		var firstImage string
		productImagesDir := filepath.Join(*imagesDir, sp.Slug)
		entries, err := os.ReadDir(productImagesDir)
		if err != nil {
			log.Info("no image directory, skipping", "path", productImagesDir)
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				fullPath := filepath.Join(productImagesDir, entry.Name())
				ilog := log.With("file", entry.Name())
				objectName, err := uploadImage(mc, sp.Slug, fullPath)
				if err != nil {
					ilog.Error("upload image", "err", err)
				} else {
					ilog.Info("image uploaded", "object", objectName)
					if firstImage == "" {
						firstImage = objectName // e.g. "products/sole-wallet/front.jpg"
					}
				}
			}
		}

		// Save the relative path of the first image as the product's primary image.
		if firstImage != "" {
			pstatus, pbody, err := cli.doJSON(http.MethodPatch,
				fmt.Sprintf("/admin/products/%d", created.ID),
				map[string]string{"primaryImageUrl": firstImage},
				nil)
			if err != nil || pstatus != http.StatusOK {
				log.Error("set primaryImageUrl failed", "status", pstatus, "body", string(pbody), "err", err)
			} else {
				log.Info("primaryImageUrl saved", "status", pstatus, "path", firstImage)
			}
		}

		seeded++
	}

	slog.Info("seed complete", "seeded", seeded, "skipped", skipped, "failed", failed)
	if failed > 0 {
		os.Exit(1)
	}
}
