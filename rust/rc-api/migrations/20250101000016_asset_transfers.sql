-- Step 4: 过户记录表
-- 对齐基线变更：过户是核心协议能力，需要完整记录
-- 参见 docs/foundation/domain-model.md §2.9 Transfer

CREATE TABLE IF NOT EXISTS asset_transfers (
    transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    from_user_id TEXT NOT NULL,
    to_user_id TEXT NOT NULL,

    -- 过户费用（可选，后续扩展）
    transfer_fee DECIMAL(10, 2),
    brand_share DECIMAL(10, 2),
    platform_share DECIMAL(10, 2),

    trace_id TEXT NOT NULL,
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_asset ON asset_transfers(asset_id);
CREATE INDEX IF NOT EXISTS idx_transfers_from_user ON asset_transfers(from_user_id);
CREATE INDEX IF NOT EXISTS idx_transfers_to_user ON asset_transfers(to_user_id);
