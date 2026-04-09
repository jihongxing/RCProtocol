-- 005_brand_api_keys.sql
-- 品牌 API Key 管理表

CREATE TABLE IF NOT EXISTS brand_api_keys (
    key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(org_id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_brand_api_keys_org_id ON brand_api_keys(org_id);
CREATE INDEX IF NOT EXISTS idx_brand_api_keys_status ON brand_api_keys(status);

COMMENT ON TABLE brand_api_keys IS '品牌 API Key 管理表，用于品牌自助对接';
COMMENT ON COLUMN brand_api_keys.key_hash IS 'bcrypt 哈希后的 API Key';
COMMENT ON COLUMN brand_api_keys.last_used_at IS '最后使用时间，用于审计';
