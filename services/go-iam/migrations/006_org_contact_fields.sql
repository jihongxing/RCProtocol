-- 006_org_contact_fields.sql
-- 品牌极简化注册：新增联系人字段

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS contact_email TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS contact_phone TEXT;

COMMENT ON COLUMN organizations.contact_email IS '品牌联系邮箱（品牌极简化注册必填）';
COMMENT ON COLUMN organizations.contact_phone IS '品牌联系电话（品牌极简化注册必填）';
