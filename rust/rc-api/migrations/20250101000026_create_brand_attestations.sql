-- Step 9: create brand_attestations table for brand trust root bridge
CREATE TABLE IF NOT EXISTS brand_attestations (
    attestation_id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    asset_commitment_id TEXT NOT NULL,
    statement TEXT NOT NULL,
    key_id TEXT NOT NULL,
    canonical_payload JSONB NOT NULL,
    signature TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(asset_commitment_id, statement)
);

CREATE INDEX IF NOT EXISTS idx_brand_attestations_brand_id
    ON brand_attestations(brand_id);

CREATE INDEX IF NOT EXISTS idx_brand_attestations_commitment
    ON brand_attestations(asset_commitment_id);
