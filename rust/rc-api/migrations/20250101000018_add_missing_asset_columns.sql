-- Add missing columns to assets table to match 001_init.sql schema

-- External SKU mapping
ALTER TABLE assets ADD COLUMN IF NOT EXISTS external_product_id VARCHAR(255);
ALTER TABLE assets ADD COLUMN IF NOT EXISTS external_product_name VARCHAR(512);
ALTER TABLE assets ADD COLUMN IF NOT EXISTS external_product_url TEXT;

-- Ownership
ALTER TABLE assets ADD COLUMN IF NOT EXISTS owner_id VARCHAR(64);

-- Key rotation tracking
ALTER TABLE assets ADD COLUMN IF NOT EXISTS key_epoch INT NOT NULL DEFAULT 0;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_key_rotation_at TIMESTAMPTZ;

-- Additional timestamps
ALTER TABLE assets ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS sold_at TIMESTAMPTZ;

-- Metadata
ALTER TABLE assets ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

-- Add state constraint if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_asset_state'
        AND conrelid = 'assets'::regclass
    ) THEN
        ALTER TABLE assets ADD CONSTRAINT chk_asset_state CHECK (current_state IN (
            'PreMinted', 'FactoryLogged', 'Unassigned', 'RotatingKeys',
            'EntangledPending', 'Activated', 'LegallySold', 'Transferred',
            'Consumed', 'Legacy', 'Tampered', 'Compromised', 'Destructed', 'Disputed'
        ));
    END IF;
END $$;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_assets_owner_id ON assets(owner_id);
CREATE INDEX IF NOT EXISTS idx_assets_external_product_id ON assets(external_product_id);
CREATE INDEX IF NOT EXISTS idx_assets_key_epoch ON assets(key_epoch);
