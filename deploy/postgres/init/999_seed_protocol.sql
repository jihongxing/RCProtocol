-- =============================================
-- RCProtocol 协议主库种子数据
-- 目标库：rcprotocol
-- 所有 INSERT 使用 ON CONFLICT 保证幂等
-- 生成时间：2026-04-07
-- =============================================

-- 品牌
INSERT INTO brands (brand_id, brand_name, status, created_at)
VALUES
  ('brand-luxe', 'Luxe 奢侈品品牌', 'Active', NOW()),
  ('brand-demo', 'RC Demo Brand', 'Active', NOW())
ON CONFLICT (brand_id) DO NOTHING;

-- 批次
INSERT INTO batches (batch_id, brand_id, created_at)
VALUES
  ('batch-luxe-001', 'brand-luxe', NOW()),
  ('batch-demo-001', 'brand-demo', NOW())
ON CONFLICT (batch_id) DO NOTHING;

-- 工厂会话
INSERT INTO factory_sessions (session_id, batch_id, created_at)
VALUES
  ('session-luxe-001', 'batch-luxe-001', NOW()),
  ('session-demo-001', 'batch-demo-001', NOW())
ON CONFLICT (session_id) DO NOTHING;

-- 资产（覆盖各主要状态）
INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, owner_id,
                    external_product_id, external_product_name, external_product_url, epoch, created_at)
VALUES
  -- 待盲扫资产（PreMinted）
  ('asset-preminted-001', 'brand-luxe', 'UID-TEST-0001', 'PreMinted', NULL, NULL, NULL, NULL, NULL, 0, NOW()),
  ('asset-preminted-002', 'brand-luxe', 'UID-TEST-0002', 'PreMinted', NULL, NULL, NULL, NULL, NULL, 0, NOW()),
  ('asset-preminted-003', 'brand-luxe', 'UID-TEST-0003', 'PreMinted', NULL, NULL, NULL, NULL, NULL, 0, NOW()),

  -- 已盲扫（FactoryLogged）
  ('asset-factorylogged-001', 'brand-luxe', 'UID-TEST-0010', 'FactoryLogged', NULL, NULL, NULL, NULL, NULL, 0, NOW()),

  -- 待分配（Unassigned）
  ('asset-unassigned-001', 'brand-luxe', 'UID-TEST-0020', 'Unassigned', NULL, NULL, NULL, NULL, NULL, 0, NOW()),

  -- 已激活（Activated）
  ('asset-activated-001', 'brand-luxe', 'UID-TEST-0030', 'Activated', NULL, NULL,
   'SKU-LUXE-001', 'Luxe经典手袋', 'https://www.luxe-brand.com/products/classic-handbag', 0, NOW()),
  ('asset-activated-002', 'brand-luxe', 'UID-TEST-0031', 'Activated', NULL, NULL,
   'SKU-LUXE-002', 'Luxe限量腕表', 'https://www.luxe-brand.com/products/limited-watch', 0, NOW()),

  -- 已合法售出（LegallySold）
  ('asset-legallysold-001', 'brand-luxe', 'UID-TEST-0040', 'LegallySold', NULL, 'user-consumer-001',
   'SKU-LUXE-001', 'Luxe经典手袋', 'https://www.luxe-brand.com/products/classic-handbag', 0, NOW()),

  -- 已过户（Transferred）
  ('asset-transferred-001', 'brand-luxe', 'UID-TEST-0050', 'Transferred', NULL, 'user-consumer-002',
   'SKU-LUXE-001', 'Luxe经典手袋', 'https://www.luxe-brand.com/products/classic-handbag', 0, NOW()),

  -- 争议中（Disputed）
  ('asset-disputed-001', 'brand-luxe', 'UID-TEST-0060', 'Disputed', 'LegallySold', 'user-consumer-001',
   'SKU-LUXE-001', 'Luxe经典手袋', 'https://www.luxe-brand.com/products/classic-handbag', 0, NOW()),

  -- 终态：已消费（Consumed）
  ('asset-consumed-001', 'brand-luxe', 'UID-TEST-0070', 'Consumed', NULL, NULL,
   'SKU-LUXE-002', 'Luxe限量腕表', 'https://www.luxe-brand.com/products/limited-watch', 0, NOW()),

  -- 终态：传承遗珍（Legacy）
  ('asset-legacy-001', 'brand-luxe', 'UID-TEST-0080', 'Legacy', 'Transferred', NULL,
   'SKU-LUXE-002', 'Luxe限量腕表', 'https://www.luxe-brand.com/products/limited-watch', 0, NOW()),

  -- 验真测试资产（合法 7 字节 hex UID）
  ('asset-verify-001', 'brand-luxe', '04A31B2C3D4E5F', 'Activated', NULL, NULL,
   'SKU-LUXE-001', 'Luxe经典手袋', 'https://www.luxe-brand.com/products/classic-handbag', 0, NOW()),

  -- Demo 品牌资产
  ('asset-demo-001', 'brand-demo', 'UID-DEMO-0001', 'Activated', NULL, NULL,
   'SKU-DEMO-001', 'Demo产品', 'https://demo.rcprotocol.dev/products/demo-001', 0, NOW())
ON CONFLICT (asset_id) DO NOTHING;

-- 母卡凭证
INSERT INTO authority_devices (authority_id, authority_uid, authority_type, brand_id, status,
                               key_epoch, bound_user_id, created_by, created_at)
VALUES
  ('a0000000-0000-0000-0000-000000000001', 'VAUTH-LUXE-0030', 'VIRTUAL_QR', 'brand-luxe', 'Active',
   0, NULL, 'user-brand-001', NOW()),
  ('a0000000-0000-0000-0000-000000000002', 'VAUTH-LUXE-0040', 'VIRTUAL_APP', 'brand-luxe', 'Active',
   0, 'user-consumer-001', 'user-brand-001', NOW()),
  ('a0000000-0000-0000-0000-000000000003', 'VAUTH-LUXE-0050', 'VIRTUAL_APP', 'brand-luxe', 'Active',
   0, 'user-consumer-002', 'user-brand-001', NOW()),
  ('a0000000-0000-0000-0000-000000000004', '04F10A2B3C4D5E', 'PHYSICAL_NFC', 'brand-luxe', 'Active',
   0, NULL, 'user-brand-001', NOW())
ON CONFLICT (authority_id) DO NOTHING;

-- 母子绑定
INSERT INTO asset_entanglements (entanglement_id, asset_id, child_uid, authority_id, authority_uid,
                                  entanglement_state, bound_by, created_at)
VALUES
  ('e0000000-0000-0000-0000-000000000001', 'asset-activated-001', 'UID-TEST-0030',
   'a0000000-0000-0000-0000-000000000001', 'VAUTH-LUXE-0030', 'Active', 'user-brand-001', NOW()),
  ('e0000000-0000-0000-0000-000000000002', 'asset-legallysold-001', 'UID-TEST-0040',
   'a0000000-0000-0000-0000-000000000002', 'VAUTH-LUXE-0040', 'Active', 'user-brand-001', NOW()),
  ('e0000000-0000-0000-0000-000000000003', 'asset-transferred-001', 'UID-TEST-0050',
   'a0000000-0000-0000-0000-000000000003', 'VAUTH-LUXE-0050', 'Active', 'user-brand-001', NOW())
ON CONFLICT (entanglement_id) DO NOTHING;

-- 过户记录
INSERT INTO asset_transfers (transfer_id, asset_id, from_user_id, to_user_id, trace_id, created_at)
VALUES
  ('t0000000-0000-0000-0000-000000000001', 'asset-transferred-001', 'user-consumer-001', 'user-consumer-002',
   'trace-transfer-001', NOW())
ON CONFLICT (transfer_id) DO NOTHING;

-- 审计日志（示例数据）
INSERT INTO audit_logs (log_id, trace_id, asset_id, action, actor_id, actor_role,
                        from_state, to_state, metadata, created_at)
VALUES
  ('log-0000000001', 'trace-001', 'asset-activated-001', 'ActivateConfirm', 'user-brand-001', 'Brand',
   'Unassigned', 'Activated', '{"external_product_id":"SKU-LUXE-001"}', NOW() - INTERVAL '10 days'),
  ('log-0000000002', 'trace-002', 'asset-legallysold-001', 'LegalSell', 'user-brand-001', 'Brand',
   'Activated', 'LegallySold', '{"buyer_id":"user-consumer-001"}', NOW() - INTERVAL '5 days'),
  ('log-0000000003', 'trace-transfer-001', 'asset-transferred-001', 'Transfer', 'user-consumer-001', 'Consumer',
   'LegallySold', 'Transferred', '{"to_user_id":"user-consumer-002"}', NOW() - INTERVAL '2 days'),
  ('log-0000000004', 'trace-003', 'asset-disputed-001', 'Freeze', 'user-moderator-001', 'Moderator',
   'LegallySold', 'Disputed', '{"reason":"用户举报疑似假货"}', NOW() - INTERVAL '1 day')
ON CONFLICT (log_id) DO NOTHING;
