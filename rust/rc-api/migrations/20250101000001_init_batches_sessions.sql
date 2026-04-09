CREATE TABLE IF NOT EXISTS batches (
    batch_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS factory_sessions (
    session_id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES batches(batch_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
