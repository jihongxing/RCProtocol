-- Step 4: 母子绑定关系表（Asset Entanglements）
-- 对齐基线变更：授权绑定关系支持虚拟和物理母卡
-- 参见 docs/foundation/domain-model.md §2.5 Entanglement

CREATE TABLE IF NOT EXISTS asset_entanglements (
    entanglement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    child_uid TEXT NOT NULL,

    authority_id UUID NOT NULL REFERENCES authority_devices(authority_id),
    authority_uid TEXT NOT NULL,

    entanglement_state TEXT NOT NULL DEFAULT 'Active',

    bound_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    bound_by TEXT NOT NULL,
    unbound_at TIMESTAMPTZ,
    unbound_reason TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 约束
    CONSTRAINT chk_entanglement_state CHECK (
        entanglement_state IN ('Pending', 'Active', 'Suspended', 'Broken')
    )
);

-- 同一资产同一时刻只能有一个 Active 绑定
CREATE UNIQUE INDEX IF NOT EXISTS uq_entanglement_active
  ON asset_entanglements(asset_id)
  WHERE entanglement_state = 'Active';

CREATE INDEX IF NOT EXISTS idx_entanglement_asset ON asset_entanglements(asset_id);
CREATE INDEX IF NOT EXISTS idx_entanglement_authority ON asset_entanglements(authority_id);
CREATE INDEX IF NOT EXISTS idx_entanglement_state ON asset_entanglements(entanglement_state);
