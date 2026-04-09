CREATE TABLE IF NOT EXISTS assets (
    asset_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    product_id TEXT NULL REFERENCES products(product_id),
    uid TEXT NULL,
    current_state TEXT NOT NULL,
    previous_state TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
