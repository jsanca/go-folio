package domain

import "time"

// SyncCursor encodes the position in a cursor-paginated result set.
type SyncCursor struct {
	UpdatedAt time.Time `json:"updatedAt"`
	ID        int64     `json:"id"`
}

// SyncQuery is the input for paginated sync operations.
// AfterAt and AfterID are set by the service layer after decoding Cursor.
type SyncQuery struct {
	PageSize     int
	Cursor       string     // raw base64 string from the request
	UpdatedSince *time.Time // filter records updated after this timestamp
	AfterAt      *time.Time // decoded from Cursor
	AfterID      int64      // decoded from Cursor
}

// ProductProjection is a denormalized view of a product together with all its variants and images.
type ProductProjection struct {
	Product  Product          `json:"product"`
	Variants []ProductVariant `json:"variants"`
	Images   []ProductImage   `json:"images"`
}

// ProductProjectionPage is a cursor-paginated response of product projections.
type ProductProjectionPage struct {
	Items      []ProductProjection `json:"items"`
	PageSize   int                 `json:"pageSize"`
	NextCursor string              `json:"nextCursor"`
	HasNext    bool                `json:"hasNext"`
	SyncToken  time.Time           `json:"syncToken"`
}

// VariantInventoryRecord is a lightweight view of a variant's inventory and pricing,
// enriched with the parent product code.
type VariantInventoryRecord struct {
	ProductCode   string      `json:"productCode"`
	ProductID     int64       `json:"productId"`
	VariantID     int64       `json:"variantId"`
	SKU           string      `json:"sku"`
	RetailPrice   Money       `json:"retailPrice"`
	SalePrice     *Money      `json:"salePrice,omitempty"`
	Currency      string      `json:"currency"`
	StockQuantity int         `json:"stockQuantity"`
	StockStatus   StockStatus `json:"stockStatus"`
	Active        bool        `json:"active"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	LastSyncedAt  *time.Time  `json:"lastSyncedAt,omitempty"`
}

// VariantInventoryPage is a cursor-paginated response of variant inventory records.
type VariantInventoryPage struct {
	Items      []VariantInventoryRecord `json:"items"`
	PageSize   int                      `json:"pageSize"`
	NextCursor string                   `json:"nextCursor"`
	HasNext    bool                     `json:"hasNext"`
	SyncToken  time.Time                `json:"syncToken"`
}
