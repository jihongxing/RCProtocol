DO $$ BEGIN
    CREATE TYPE protocol_role AS ENUM ('Platform', 'Brand', 'Factory', 'Consumer', 'Moderator');
EXCEPTION WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS positions (
    position_id   UUID PRIMARY KEY,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    position_name TEXT NOT NULL,
    protocol_role protocol_role NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_positions_org_id ON positions(org_id);
