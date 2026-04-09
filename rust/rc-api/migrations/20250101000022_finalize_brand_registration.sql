-- Step 5: finalize brand registration schema for spec-04
-- Align runtime schema with routes/db code that expects:
--   brands.contact_email, brands.industry, brands.status, brands.updated_at
--   api_keys table with lifecycle fields

ALTER TABLE brands
  ADD COLUMN IF NOT EXISTS contact_email TEXT,
  ADD COLUMN IF NOT EXISTS industry TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'Active';

UPDATE brands
SET contact_email = COALESCE(contact_email, brand_id || '@placeholder.local')
WHERE contact_email IS NULL;

UPDATE brands
SET industry = COALESCE(industry, 'Other')
WHERE industry IS NULL;

ALTER TABLE brands
  ALTER COLUMN contact_email SET NOT NULL,
  ALTER COLUMN industry SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_brands_contact_email
  ON brands(contact_email);

CREATE INDEX IF NOT EXISTS idx_brands_status
  ON brands(status);

CREATE INDEX IF NOT EXISTS idx_brands_created_at
  ON brands(created_at DESC);

CREATE TABLE IF NOT EXISTS api_keys (
    key_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_api_keys_brand_id
  ON api_keys(brand_id);

CREATE INDEX IF NOT EXISTS idx_api_keys_status
  ON api_keys(status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash
  ON api_keys(key_hash);
