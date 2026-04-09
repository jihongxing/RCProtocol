DO $$ BEGIN
    CREATE TYPE org_type AS ENUM ('platform', 'brand', 'factory');
EXCEPTION WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS organizations (
    org_id        UUID PRIMARY KEY,
    org_name      TEXT NOT NULL,
    org_type      org_type NOT NULL,
    parent_org_id UUID REFERENCES organizations(org_id),
    brand_id      TEXT,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_organizations_org_type ON organizations(org_type);
CREATE INDEX IF NOT EXISTS idx_organizations_brand_id ON organizations(brand_id);
