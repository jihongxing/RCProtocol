CREATE TABLE IF NOT EXISTS user_org_positions (
    user_id     UUID NOT NULL REFERENCES users(user_id),
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    position_id UUID NOT NULL REFERENCES positions(position_id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, org_id)
);
CREATE INDEX IF NOT EXISTS idx_uop_user_id ON user_org_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_uop_org_id ON user_org_positions(org_id);
