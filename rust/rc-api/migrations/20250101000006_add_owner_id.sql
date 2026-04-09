-- 钱包快照查询（WHERE owner_id = $1）和资产列表 read-through 均依赖此列
ALTER TABLE assets ADD COLUMN IF NOT EXISTS owner_id TEXT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_owner_id ON assets(owner_id);
