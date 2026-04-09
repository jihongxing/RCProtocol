-- Step 4: 品牌极简化 + API Key 支持
-- 对齐基线变更：品牌注册从 30+ 字段简化为 3 个核心字段
-- 参见 docs/foundation/domain-model.md §3.2, docs/foundation/api-and-service-boundaries.md §10

-- brands 表增加 API Key 和可选展示字段
ALTER TABLE brands
  ADD COLUMN IF NOT EXISTS api_key_hash TEXT,
  ADD COLUMN IF NOT EXISTS brand_logo TEXT,
  ADD COLUMN IF NOT EXISTS brand_website TEXT,
  ADD COLUMN IF NOT EXISTS webhook_url TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'Active',
  ADD COLUMN IF NOT EXISTS created_by TEXT;

-- API Key hash 唯一索引（用于认证查找）
CREATE UNIQUE INDEX IF NOT EXISTS uq_brands_api_key_hash
  ON brands(api_key_hash) WHERE api_key_hash IS NOT NULL;

-- brands.status 合法值约束
ALTER TABLE brands
  ADD CONSTRAINT chk_brands_status CHECK (
    status IN ('Active', 'Suspended')
  );
