-- DEPRECATED: 此文件不再被 docker-compose 使用。
-- Schema 现由 rc-api 启动时通过 sqlx::migrate!() 自动管理。
-- 参见 rust/rc-api/migrations/
-- 保留此文件作为原始 schema 参考。
--
-- 注意：此 schema 已过时。当前基线变更包括：
-- - brands 表极简化（3 字段 + API Key）
-- - products 表不再使用（改为 assets 上的外部 SKU 映射）
-- - 新增 authority_devices 表（虚拟/物理母卡）
-- - 新增 asset_entanglements 表（母子绑定）
-- - 新增 asset_transfers 表（过户记录）
-- 详见 migrations 000012 ~ 000016

-- RCProtocol PostgreSQL bootstrap schema
-- Current purpose: establish authoritative persistence skeleton for protocol truth.

CREATE TABLE IF NOT EXISTS brands (
    brand_id TEXT PRIMARY KEY,
    brand_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    product_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    product_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS batches (
    batch_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS factory_sessions (
    session_id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES batches(batch_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assets (
    asset_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id),
    product_id TEXT NULL REFERENCES products(product_id),
    uid TEXT NULL,
    current_state TEXT NOT NULL,
    previous_state TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS asset_state_events (
    event_id UUID PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    action TEXT NOT NULL,
    from_state TEXT NULL,
    to_state TEXT NOT NULL,
    trace_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    actor_org TEXT NULL,
    idempotency_key TEXT NOT NULL,
    approval_id TEXT NULL,
    policy_version TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS idempotency_records (
    idempotency_key TEXT PRIMARY KEY,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assets_brand_id ON assets(brand_id);
CREATE INDEX IF NOT EXISTS idx_assets_current_state ON assets(current_state);
CREATE INDEX IF NOT EXISTS idx_asset_state_events_asset_id ON asset_state_events(asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_state_events_trace_id ON asset_state_events(trace_id);
