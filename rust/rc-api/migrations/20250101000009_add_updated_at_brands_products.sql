-- M1: brands/products 表添加 updated_at 列 (Bug 1.8)
-- 代码查询 updated_at 但 Migration 中只有 created_at，导致 500
ALTER TABLE brands
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
