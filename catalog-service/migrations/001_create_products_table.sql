-- Migration: 001_create_products_table
-- Target: SQLite

CREATE TABLE IF NOT EXISTS products (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    sku                  TEXT    NOT NULL UNIQUE,
    external_system_id   TEXT    NOT NULL DEFAULT '',
    title                TEXT    NOT NULL,
    slug                 TEXT    NOT NULL UNIQUE,
    short_description    TEXT    NOT NULL DEFAULT '',
    description          TEXT    NOT NULL DEFAULT '',
    category             TEXT    NOT NULL DEFAULT '',
    tags                 TEXT    NOT NULL DEFAULT '[]',
    main_image_url       TEXT    NOT NULL DEFAULT '',
    retail_price_cents   INTEGER NOT NULL CHECK (retail_price_cents >= 0),
    sale_price_cents     INTEGER          CHECK (sale_price_cents IS NULL OR sale_price_cents >= 0),
    currency             TEXT    NOT NULL,
    stock_quantity       INTEGER NOT NULL DEFAULT 0,
    stock_status         TEXT    NOT NULL CHECK (stock_status IN ('IN_STOCK', 'LOW_STOCK', 'OUT_OF_STOCK')),
    warehouse_code       TEXT    NOT NULL DEFAULT '',
    active               INTEGER NOT NULL DEFAULT 1,
    created_at           DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at           DATETIME NOT NULL DEFAULT (datetime('now')),
    last_synced_at       DATETIME
);

CREATE INDEX IF NOT EXISTS idx_products_sku          ON products (sku);
CREATE INDEX IF NOT EXISTS idx_products_slug         ON products (slug);
CREATE INDEX IF NOT EXISTS idx_products_category     ON products (category);
CREATE INDEX IF NOT EXISTS idx_products_active       ON products (active);
CREATE INDEX IF NOT EXISTS idx_products_stock_status ON products (stock_status);
