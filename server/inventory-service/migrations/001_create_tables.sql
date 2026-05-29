CREATE TABLE IF NOT EXISTS stock (
    sku       TEXT    PRIMARY KEY,
    available INTEGER NOT NULL DEFAULT 0,
    reserved  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS reservations (
    id         TEXT        PRIMARY KEY,
    sku        TEXT        NOT NULL REFERENCES stock(sku),
    quantity   INTEGER     NOT NULL,
    order_id   TEXT        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
