-- Step 4: 资产外部 SKU 映射
-- 对齐基线变更：不管理 SKU，只存储"资产与外部 SKU 的映射关系"
-- 参见 docs/foundation/domain-model.md §2.11 External Product Mapping

-- assets 表增加外部 SKU 映射字段
ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS external_product_id TEXT,
  ADD COLUMN IF NOT EXISTS external_product_name TEXT,
  ADD COLUMN IF NOT EXISTS external_product_url TEXT;

-- 外部 SKU 映射索引（品牌方按 SKU 查询资产）
CREATE INDEX IF NOT EXISTS idx_assets_external_product
  ON assets(external_product_id) WHERE external_product_id IS NOT NULL;

-- assets 表去掉对 products 表的外键依赖
-- product_id 列保留但不再使用，避免破坏已有数据
-- 新资产不再写入 product_id，改用 external_product_id
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_product_id_fkey;
