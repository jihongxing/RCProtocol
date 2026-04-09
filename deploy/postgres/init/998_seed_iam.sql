-- =============================================
-- RCProtocol IAM 种子数据
-- 目标库：rcprotocol_iam
-- 所有 INSERT 使用 ON CONFLICT 保证幂等
-- 生成时间：2026-04-07
-- =============================================

-- 组织
INSERT INTO organizations (org_id, org_name, org_type, brand_id, contact_email, contact_phone, status, created_at)
VALUES
  ('org-platform-001', 'RCProtocol 平台运营', 'Platform', NULL, 'platform@test.rcprotocol.dev', '13800000001', 'active', NOW()),
  ('org-brand-luxe', 'Luxe 奢侈品品牌', 'Brand', 'brand-luxe', 'brand@test.rcprotocol.dev', '13800000002', 'active', NOW()),
  ('org-brand-demo', 'RC Demo Brand', 'Brand', 'brand-demo', 'demo@test.rcprotocol.dev', '13800000004', 'active', NOW()),
  ('org-factory-shenzhen', '深圳标签工厂', 'Factory', NULL, 'factory@test.rcprotocol.dev', '13800000003', 'active', NOW())
ON CONFLICT (org_id) DO NOTHING;

-- 岗位
INSERT INTO positions (position_id, org_id, position_name, protocol_role, created_at)
VALUES
  ('pos-platform-admin', 'org-platform-001', '平台管理员', 'Platform', NOW()),
  ('pos-platform-moderator', 'org-platform-001', '审核员', 'Moderator', NOW()),
  ('pos-brand-admin', 'org-brand-luxe', '品牌管理员', 'Brand', NOW()),
  ('pos-brand-demo-admin', 'org-brand-demo', 'Demo品牌管理员', 'Brand', NOW()),
  ('pos-factory-operator', 'org-factory-shenzhen', '工厂操作员', 'Factory', NOW())
ON CONFLICT (position_id) DO NOTHING;

-- 用户（密码使用 bcrypt cost=12）
-- Admin@2026   -> $2b$12$I2rGwanz46Cb5sAU0924PO.713QMaeq4EJNRktgyDQUF6oLKatNtK
-- Mod@2026     -> $2b$12$tMG8iSQPZHWcQsgz/CCLZ.sms67nIerkaF356LYh/nGvgNAv6kHYW
-- Brand@2026   -> $2b$12$JLPbDraWKrB/jYMzvexXPutTgPtbPofb0NPXq1FHweJXH2Dcb2QDC
-- Factory@2026 -> $2b$12$TwM7wvYuZqvp1TQNGv7.teMEfvQjSbxn0A01IyOoNukM3egdxHcFW
-- Consumer@2026-> $2b$12$kQ3kqP1vnJNhP7SRXeSsteZYSbC909RHPGI4Yi.76Sh7CoeDGsi6G

INSERT INTO users (user_id, email, password_hash, display_name, status, created_at)
VALUES
  ('user-admin-001', 'admin@test.rcprotocol.dev',
   '$2b$12$I2rGwanz46Cb5sAU0924PO.713QMaeq4EJNRktgyDQUF6oLKatNtK',
   '平台管理员张三', 'active', NOW()),
  ('user-moderator-001', 'moderator@test.rcprotocol.dev',
   '$2b$12$tMG8iSQPZHWcQsgz/CCLZ.sms67nIerkaF356LYh/nGvgNAv6kHYW',
   '审核员李四', 'active', NOW()),
  ('user-brand-001', 'brand@test.rcprotocol.dev',
   '$2b$12$JLPbDraWKrB/jYMzvexXPutTgPtbPofb0NPXq1FHweJXH2Dcb2QDC',
   'Luxe品牌运营王五', 'active', NOW()),
  ('user-brand-002', 'demo@test.rcprotocol.dev',
   '$2b$12$JLPbDraWKrB/jYMzvexXPutTgPtbPofb0NPXq1FHweJXH2Dcb2QDC',
   'Demo品牌运营', 'active', NOW()),
  ('user-factory-001', 'factory@test.rcprotocol.dev',
   '$2b$12$TwM7wvYuZqvp1TQNGv7.teMEfvQjSbxn0A01IyOoNukM3egdxHcFW',
   '工厂操作员赵六', 'active', NOW()),
  ('user-consumer-001', 'consumer1@test.rcprotocol.dev',
   '$2b$12$kQ3kqP1vnJNhP7SRXeSsteZYSbC909RHPGI4Yi.76Sh7CoeDGsi6G',
   '消费者测试用户A', 'active', NOW()),
  ('user-consumer-002', 'consumer2@test.rcprotocol.dev',
   '$2b$12$kQ3kqP1vnJNhP7SRXeSsteZYSbC909RHPGI4Yi.76Sh7CoeDGsi6G',
   '消费者测试用户B', 'active', NOW())
ON CONFLICT (user_id) DO NOTHING;

-- 用户-组织-岗位绑定
INSERT INTO user_org_positions (user_id, org_id, position_id, created_at)
VALUES
  ('user-admin-001', 'org-platform-001', 'pos-platform-admin', NOW()),
  ('user-moderator-001', 'org-platform-001', 'pos-platform-moderator', NOW()),
  ('user-brand-001', 'org-brand-luxe', 'pos-brand-admin', NOW()),
  ('user-brand-002', 'org-brand-demo', 'pos-brand-demo-admin', NOW()),
  ('user-factory-001', 'org-factory-shenzhen', 'pos-factory-operator', NOW())
ON CONFLICT (user_id, org_id) DO NOTHING;

-- 品牌 API Key
-- Active Key: brand_org-brand-luxe_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4
--   -> $2b$12$okgq.gB/sPgJDhqX8t5QZuTSwnfCzVIYe2V1IOY1s9ctSqWK2/Chy
-- Revoked Key: brand_org-brand-luxe_revoked_key_for_testing_only
--   -> $2b$12$QQMA5A43Xd1hSA/0tjm.c.5KhzzL.66DzEoLE.lpYo63sBuy37w8C

INSERT INTO brand_api_keys (key_id, org_id, key_hash, description, status, created_at, last_used_at, revoked_at)
VALUES
  ('apikey-luxe-001', 'org-brand-luxe',
   '$2b$12$okgq.gB/sPgJDhqX8t5QZuTSwnfCzVIYe2V1IOY1s9ctSqWK2/Chy',
   'Luxe 品牌集成测试 Key', 'active', NOW(), NULL, NULL),
  ('apikey-luxe-002', 'org-brand-luxe',
   '$2b$12$QQMA5A43Xd1hSA/0tjm.c.5KhzzL.66DzEoLE.lpYo63sBuy37w8C',
   'Luxe 品牌已撤销的测试 Key', 'revoked', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', NOW() - INTERVAL '1 day')
ON CONFLICT (key_id) DO NOTHING;
