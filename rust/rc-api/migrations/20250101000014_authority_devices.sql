-- Step 4: 母卡凭证表（Authority Devices）
-- 对齐基线变更：虚拟母卡为默认形态，物理母卡高端可选
-- 参见 docs/foundation/domain-model.md §2.3, docs/foundation/security-model.md §5

CREATE TABLE IF NOT EXISTS authority_devices (
    authority_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 母卡唯一标识（物理卡为 chip UID，虚拟卡为系统生成）
    authority_uid TEXT NOT NULL UNIQUE,

    -- 母卡类型：协议核心层对形态无感知，校验逻辑根据此字段分发
    authority_type TEXT NOT NULL,

    brand_id TEXT NOT NULL REFERENCES brands(brand_id),

    status TEXT NOT NULL DEFAULT 'Active',

    -- 密钥轮换版本
    key_epoch INTEGER NOT NULL DEFAULT 0,

    -- 物理卡专属字段（虚拟卡为 NULL）
    physical_chip_uid TEXT,
    last_known_ctr INTEGER,

    -- 虚拟卡专属字段（物理卡为 NULL）
    virtual_credential_hash TEXT,
    bound_user_id TEXT,

    -- 审计字段
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,

    -- 约束
    CONSTRAINT chk_authority_type CHECK (
        authority_type IN ('PHYSICAL_NFC', 'VIRTUAL_QR', 'VIRTUAL_APP', 'VIRTUAL_BIOMETRIC')
    ),
    CONSTRAINT chk_authority_status CHECK (
        status IN ('Active', 'Suspended', 'Revoked', 'Replaced')
    )
);

CREATE INDEX IF NOT EXISTS idx_authority_brand ON authority_devices(brand_id);
CREATE INDEX IF NOT EXISTS idx_authority_type ON authority_devices(authority_type);
CREATE INDEX IF NOT EXISTS idx_authority_user ON authority_devices(bound_user_id)
  WHERE bound_user_id IS NOT NULL;
