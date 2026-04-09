-- Step 8: bridge current assets to asset commitments
ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS asset_commitment_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_assets_asset_commitment_id
  ON assets(asset_commitment_id);

ALTER TABLE asset_state_events
  ADD COLUMN IF NOT EXISTS asset_commitment_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_state_events_asset_commitment_id
  ON asset_state_events(asset_commitment_id);
