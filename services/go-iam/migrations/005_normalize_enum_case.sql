-- M9: 统一枚举命名风格 — org_type 改为首字母大写（与 protocol_role 一致）
-- 步骤: 1) 更新现有数据  2) 重建枚举类型

-- Step 1: 更新现有数据到首字母大写
UPDATE organizations SET org_type = 'Platform' WHERE org_type = 'platform';
UPDATE organizations SET org_type = 'Brand'    WHERE org_type = 'brand';
UPDATE organizations SET org_type = 'Factory'  WHERE org_type = 'factory';

-- Step 2: 将列临时改为 TEXT
ALTER TABLE organizations ALTER COLUMN org_type TYPE TEXT;

-- Step 3: 删除旧枚举并重建
DROP TYPE IF EXISTS org_type;
CREATE TYPE org_type AS ENUM ('Platform', 'Brand', 'Factory');

-- Step 4: 将列改回枚举类型
ALTER TABLE organizations ALTER COLUMN org_type TYPE org_type USING org_type::org_type;
