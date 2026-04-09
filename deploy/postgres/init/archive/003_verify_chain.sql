-- Spec-05: 验真链路增量迁移
-- 新增验真相关列和审计事件表，不修改 001_init.sql 和 002_seed.sql

-- assets 表新增验真相关列
ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_verified_ctr INTEGER DEFAULT NULL;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS epoch INTEGER NOT NULL DEFAULT 0;

-- UID 索引（验真按 uid 查询）
CREATE INDEX IF NOT EXISTS idx_assets_uid ON assets(uid);

-- 验真审计事件表
CREATE TABLE IF NOT EXISTS verification_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uid TEXT NOT NULL,
    asset_id TEXT,
    ctr INTEGER NOT NULL,
    verification_status TEXT NOT NULL,
    risk_flags TEXT[] NOT NULL DEFAULT '{}',
    cmac_valid BOOLEAN NOT NULL,
    client_ip TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_events_uid ON verification_events(uid);
CREATE INDEX IF NOT EXISTS idx_verification_events_created_at ON verification_events(created_at);
