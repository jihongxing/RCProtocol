-- Step 7: create asset_commitments table for protocol commitment bridge
CREATE TABLE IF NOT EXISTS asset_commitments (
    commitment_id TEXT PRIMARY KEY,
    payload_version TEXT NOT NULL,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    asset_uid TEXT NOT NULL,
    chip_binding TEXT NOT NULL,
    epoch INTEGER NOT NULL,
    metadata_hash TEXT NOT NULL,
    canonical_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_asset_commitments_brand_uid_epoch
    ON asset_commitments(brand_id, asset_uid, epoch);

CREATE INDEX IF NOT EXISTS idx_asset_commitments_asset_uid
    ON asset_commitments(asset_uid);
