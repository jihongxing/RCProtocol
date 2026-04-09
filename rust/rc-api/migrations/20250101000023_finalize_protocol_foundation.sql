-- Step 6: finalize protocol foundation schema to match runtime code
-- This migration closes the remaining gaps between runtime SQL usage,
-- historical migrations, and test helper bootstrap logic.

-- -------------------------------------------------------------------
-- batches table: align with db/batches.rs
-- -------------------------------------------------------------------
ALTER TABLE batches
  ADD COLUMN IF NOT EXISTS batch_name TEXT,
  ADD COLUMN IF NOT EXISTS factory_id TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'Open',
  ADD COLUMN IF NOT EXISTS expected_count INTEGER,
  ADD COLUMN IF NOT EXISTS actual_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_batches_brand_id ON batches(brand_id);
CREATE INDEX IF NOT EXISTS idx_batches_status ON batches(status);

-- -------------------------------------------------------------------
-- verification_events: align with db/verification.rs runtime insert shape
-- runtime expects: uid, asset_id, ctr, verification_status, risk_flags,
-- cmac_valid, client_ip
-- -------------------------------------------------------------------
ALTER TABLE verification_events
  ADD COLUMN IF NOT EXISTS ctr INTEGER,
  ADD COLUMN IF NOT EXISTS cmac_valid BOOLEAN,
  ADD COLUMN IF NOT EXISTS client_ip TEXT;

UPDATE verification_events
SET ctr = COALESCE(ctr, ctr_received)
WHERE ctr IS NULL;

UPDATE verification_events
SET client_ip = COALESCE(client_ip, ip_address)
WHERE client_ip IS NULL;

ALTER TABLE verification_events
  ALTER COLUMN ctr SET DEFAULT 0,
  ALTER COLUMN cmac_valid SET DEFAULT FALSE;

UPDATE verification_events
SET cmac_valid = COALESCE(cmac_valid, FALSE)
WHERE cmac_valid IS NULL;

ALTER TABLE verification_events
  ALTER COLUMN ctr SET NOT NULL,
  ALTER COLUMN cmac_valid SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_verification_events_uid ON verification_events(uid);
CREATE INDEX IF NOT EXISTS idx_verification_events_status ON verification_events(verification_status);

-- -------------------------------------------------------------------
-- asset_transfers: align with db/transfers.rs runtime insert shape
-- runtime expects: asset_id, from_owner_id, to_owner_id, transfer_type,
-- idempotency_key
-- -------------------------------------------------------------------
ALTER TABLE asset_transfers
  ADD COLUMN IF NOT EXISTS from_owner_id TEXT,
  ADD COLUMN IF NOT EXISTS to_owner_id TEXT,
  ADD COLUMN IF NOT EXISTS transfer_type TEXT,
  ADD COLUMN IF NOT EXISTS idempotency_key TEXT,
  ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

UPDATE asset_transfers
SET from_owner_id = COALESCE(from_owner_id, from_user_id)
WHERE from_owner_id IS NULL;

UPDATE asset_transfers
SET to_owner_id = COALESCE(to_owner_id, to_user_id)
WHERE to_owner_id IS NULL;

UPDATE asset_transfers
SET transfer_type = COALESCE(transfer_type, 'UNKNOWN')
WHERE transfer_type IS NULL;

UPDATE asset_transfers
SET idempotency_key = COALESCE(idempotency_key, 'legacy-' || transfer_id::text)
WHERE idempotency_key IS NULL;

ALTER TABLE asset_transfers
  ALTER COLUMN from_owner_id SET NOT NULL,
  ALTER COLUMN to_owner_id SET NOT NULL,
  ALTER COLUMN transfer_type SET NOT NULL,
  ALTER COLUMN idempotency_key SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transfers_transfer_type ON asset_transfers(transfer_type);
CREATE INDEX IF NOT EXISTS idx_transfers_idempotency_key ON asset_transfers(idempotency_key);

-- -------------------------------------------------------------------
-- asset_state_events and assets helper indexes used heavily in runtime
-- -------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_assets_uid ON assets(uid);
CREATE INDEX IF NOT EXISTS idx_assets_created_at ON assets(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assets_updated_at ON assets(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_state_events_actor_role ON asset_state_events(actor_role);
