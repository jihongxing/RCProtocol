-- Step 11: extend verification_events for verification v2 attestation audit
ALTER TABLE verification_events
  ADD COLUMN IF NOT EXISTS asset_commitment_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS verification_version TEXT NULL,
  ADD COLUMN IF NOT EXISTS brand_attestation_status TEXT NULL,
  ADD COLUMN IF NOT EXISTS platform_attestation_status TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_verification_events_asset_commitment_id
  ON verification_events(asset_commitment_id);

CREATE INDEX IF NOT EXISTS idx_verification_events_verification_version
  ON verification_events(verification_version);
