-- Step 10: create platform_attestations table for protocol acceptance bridge
CREATE TABLE IF NOT EXISTS platform_attestations (
    attestation_id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    platform_id TEXT NOT NULL,
    asset_commitment_id TEXT NOT NULL,
    statement TEXT NOT NULL,
    key_id TEXT NOT NULL,
    canonical_payload JSONB NOT NULL,
    signature TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(asset_commitment_id, statement)
);

CREATE INDEX IF NOT EXISTS idx_platform_attestations_platform_id
    ON platform_attestations(platform_id);

CREATE INDEX IF NOT EXISTS idx_platform_attestations_commitment
    ON platform_attestations(asset_commitment_id);
