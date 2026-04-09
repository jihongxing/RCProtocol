-- M1: 创建 verification_events 表 (Bug 1.7)
-- 验真审计事件写入需要此表，当前缺失导致写入失败
CREATE TABLE IF NOT EXISTS verification_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    uid TEXT,
    ctr_received INTEGER,
    ctr_previous INTEGER,
    cmac_provided TEXT,
    cmac_expected TEXT,
    verification_status TEXT NOT NULL,
    risk_flags TEXT[] DEFAULT '{}',
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_events_asset_id ON verification_events(asset_id);
CREATE INDEX IF NOT EXISTS idx_verification_events_created_at ON verification_events(created_at);
