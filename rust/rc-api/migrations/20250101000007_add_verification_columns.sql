-- M1: 补全 assets 表验真相关列 (Bug 1.6)
-- assets 表缺少 last_verified_ctr 和 epoch 列，导致验真端点 500
ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS last_verified_ctr INTEGER DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS epoch INTEGER NOT NULL DEFAULT 0;
