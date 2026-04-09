-- M2: 协议表枚举约束 + brand_name 唯一 + uid 唯一 (Bug 1.41, 1.42, 1.16)
-- 当前 TEXT 列无 CHECK 约束，数据库无法拦截非法状态值

-- assets.current_state 合法值约束（14 态）
ALTER TABLE assets
  ADD CONSTRAINT chk_assets_current_state CHECK (
    current_state IN ('PreMinted','FactoryLogged','Unassigned','RotatingKeys',
      'EntangledPending','Activated','LegallySold','Transferred','Consumed',
      'Legacy','Tampered','Compromised','Destructed','Disputed')
  );

-- assets.previous_state 合法值约束（14 态 + NULL）
ALTER TABLE assets
  ADD CONSTRAINT chk_assets_previous_state CHECK (
    previous_state IS NULL OR previous_state IN ('PreMinted','FactoryLogged',
      'Unassigned','RotatingKeys','EntangledPending','Activated','LegallySold',
      'Transferred','Consumed','Legacy','Tampered','Compromised','Destructed','Disputed')
  );

-- asset_state_events.action 合法值约束（含 MarkDestructed）
ALTER TABLE asset_state_events
  ADD CONSTRAINT chk_events_action CHECK (
    action IN ('BlindLog','StockIn','ActivateRotateKeys','ActivateEntangle',
      'ActivateConfirm','LegalSell','Transfer','Consume','Legacy','Freeze',
      'Recover','MarkTampered','MarkCompromised','MarkDestructed')
  );

-- asset_state_events.actor_role 合法值约束（5 角色）
ALTER TABLE asset_state_events
  ADD CONSTRAINT chk_events_actor_role CHECK (
    actor_role IN ('Platform','Brand','Factory','Consumer','Moderator')
  );

-- assets.uid 部分唯一索引（防止同一 NFC UID 重复登记）
CREATE UNIQUE INDEX IF NOT EXISTS uq_assets_uid
  ON assets(uid) WHERE uid IS NOT NULL;

-- brands.brand_name 唯一约束
ALTER TABLE brands ADD CONSTRAINT uq_brands_name UNIQUE (brand_name);
