CREATE TABLE IF NOT EXISTS catalog_products (
    id                  BIGSERIAL    PRIMARY KEY,
    product_code        TEXT         NOT NULL UNIQUE,
    external_product_id TEXT,
    title               TEXT         NOT NULL,
    slug                TEXT         NOT NULL UNIQUE,
    short_description   TEXT         NOT NULL DEFAULT '',
    description         TEXT         NOT NULL DEFAULT '',
    additional_info     TEXT         NOT NULL DEFAULT '',
    department          TEXT         NOT NULL DEFAULT '',
    category            TEXT         NOT NULL DEFAULT '',
    subcategory         TEXT         NOT NULL DEFAULT '',
    tags                TEXT         NOT NULL DEFAULT '[]',
    base_sku            TEXT,
    active              BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_synced_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS product_variants (
    id                   BIGSERIAL   PRIMARY KEY,
    product_id           BIGINT      NOT NULL REFERENCES catalog_products(id) ON DELETE CASCADE,
    sku                  TEXT        NOT NULL UNIQUE,
    external_variant_id  TEXT,
    color_slug           TEXT        NOT NULL DEFAULT '',
    color_name           TEXT        NOT NULL DEFAULT '',
    primary_color_name   TEXT        NOT NULL DEFAULT '',
    secondary_color_name TEXT        NOT NULL DEFAULT '',
    primary_color_hex    TEXT        NOT NULL DEFAULT '',
    secondary_color_hex  TEXT        NOT NULL DEFAULT '',
    retail_price_cents   BIGINT      NOT NULL CHECK (retail_price_cents >= 0),
    sale_price_cents     BIGINT      CHECK (sale_price_cents IS NULL OR sale_price_cents >= 0),
    currency             TEXT        NOT NULL CHECK (length(currency) = 3),
    stock_quantity       INTEGER     NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    stock_status         TEXT        NOT NULL CHECK (stock_status IN ('IN_STOCK', 'LOW_STOCK', 'OUT_OF_STOCK')),
    warehouse_code       TEXT        NOT NULL DEFAULT '',
    variant_image_url    TEXT        NOT NULL DEFAULT '',
    active               BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at       TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS product_images (
    id         BIGSERIAL   PRIMARY KEY,
    product_id BIGINT      NOT NULL REFERENCES catalog_products(id),
    variant_id BIGINT      REFERENCES product_variants(id) ON DELETE CASCADE,
    url        TEXT        NOT NULL,
    alt_text   TEXT        NOT NULL DEFAULT '',
    sort_order INTEGER     NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    is_main    BOOLEAN     NOT NULL DEFAULT FALSE,
    width      INTEGER     CHECK (width IS NULL OR width > 0),
    height     INTEGER     CHECK (height IS NULL OR height > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_catalog_products_product_code ON catalog_products (product_code);
CREATE INDEX IF NOT EXISTS idx_catalog_products_slug         ON catalog_products (slug);
CREATE INDEX IF NOT EXISTS idx_catalog_products_department   ON catalog_products (department);
CREATE INDEX IF NOT EXISTS idx_catalog_products_category     ON catalog_products (category);
CREATE INDEX IF NOT EXISTS idx_catalog_products_subcategory  ON catalog_products (subcategory);
CREATE INDEX IF NOT EXISTS idx_catalog_products_active       ON catalog_products (active);

CREATE INDEX IF NOT EXISTS idx_product_variants_sku          ON product_variants (sku);
CREATE INDEX IF NOT EXISTS idx_product_variants_product_id   ON product_variants (product_id);
CREATE INDEX IF NOT EXISTS idx_product_variants_stock_status ON product_variants (stock_status);
CREATE INDEX IF NOT EXISTS idx_product_variants_active       ON product_variants (active);
CREATE INDEX IF NOT EXISTS idx_product_variants_color_slug   ON product_variants (color_slug);

CREATE INDEX IF NOT EXISTS idx_product_images_product_id     ON product_images (product_id);
CREATE INDEX IF NOT EXISTS idx_product_images_variant_id     ON product_images (variant_id);
CREATE INDEX IF NOT EXISTS idx_product_images_is_main        ON product_images (is_main);
