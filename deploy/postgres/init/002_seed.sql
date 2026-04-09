-- RCProtocol Seed Data for Local Development
-- Version: 1.0
-- Last Updated: 2026-04-08
-- Description: Test data for local development and integration testing

-- ============================================================================
-- TEST BRANDS
-- ============================================================================

-- Test Brand 1: Luxury Watch Brand
INSERT INTO brands (brand_id, brand_name, contact_email, industry, brand_key_hash, status, metadata) VALUES
    ('brand_luxury_watch', 'Luxury Watch Co.', 'contact@luxurywatch.example', 'luxury_watches', 'test_hash_luxury_watch_001', 'Active',
     '{"tier": "premium", "test": true}'::jsonb);

-- Test Brand 2: High-End Fashion
INSERT INTO brands (brand_id, brand_name, contact_email, industry, brand_key_hash, status, metadata) VALUES
    ('brand_fashion', 'Elite Fashion House', 'contact@elitefashion.example', 'fashion', 'test_hash_fashion_002', 'Active',
     '{"tier": "luxury", "test": true}'::jsonb);

-- Test Brand 3: Fine Wine
INSERT INTO brands (brand_id, brand_name, contact_email, industry, brand_key_hash, status, metadata) VALUES
    ('brand_wine', 'Premium Vineyard', 'contact@premiumvineyard.example', 'wine', 'test_hash_wine_003', 'Active',
     '{"tier": "premium", "test": true}'::jsonb);

-- ============================================================================
-- TEST API KEYS
-- ============================================================================

-- API Key for Luxury Watch Brand
-- Key format: rc_test_luxurywatch_abc123xyz (this is the actual key, hash it in production)
INSERT INTO api_keys (key_id, brand_id, key_hash, key_prefix, name, status, created_by) VALUES
    ('11111111-1111-1111-1111-111111111111',
     'brand_luxury_watch',
     'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', -- SHA-256 of 'rc_test_luxurywatch_abc123xyz'
     'rc_test_lux',
     'Development API Key',
     'Active',
     'system');

-- API Key for Fashion Brand
INSERT INTO api_keys (key_id, brand_id, key_hash, key_prefix, name, status, created_by) VALUES
    ('22222222-2222-2222-2222-222222222222',
     'brand_fashion',
     'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', -- SHA-256 of 'rc_test_fashion_def456uvw'
     'rc_test_fas',
     'Development API Key',
     'Active',
     'system');

-- ============================================================================
-- TEST WEBHOOK CONFIGS
-- ============================================================================

INSERT INTO webhook_configs (webhook_id, brand_id, url, secret, events, status) VALUES
    ('33333333-3333-3333-3333-333333333333',
     'brand_luxury_watch',
     'https://webhook.site/test-luxury-watch',
     'test_webhook_secret_luxury',
     ARRAY['asset.activated', 'asset.transferred', 'asset.disputed'],
     'Active');

INSERT INTO webhook_configs (webhook_id, brand_id, url, secret, events, status) VALUES
    ('44444444-4444-4444-4444-444444444444',
     'brand_fashion',
     'https://webhook.site/test-fashion',
     'test_webhook_secret_fashion',
     ARRAY['asset.activated', 'asset.sold', 'asset.transferred'],
     'Active');

-- ============================================================================
-- TEST BATCHES
-- ============================================================================

-- Batch 1: Luxury Watch Production Run
INSERT INTO batches (batch_id, brand_id, batch_name, factory_id, status, expected_count, metadata) VALUES
    ('55555555-5555-5555-5555-555555555555',
     'brand_luxury_watch',
     'Q1-2026-Watch-Batch-001',
     'factory_swiss_001',
     'Open',
     100,
     '{"production_line": "A1", "quality_grade": "AAA", "test": true}'::jsonb);

-- Batch 2: Fashion Collection
INSERT INTO batches (batch_id, brand_id, batch_name, factory_id, status, expected_count, metadata) VALUES
    ('66666666-6666-6666-6666-666666666666',
     'brand_fashion',
     'Spring-2026-Collection',
     'factory_italy_002',
     'Open',
     50,
     '{"collection": "Spring 2026", "category": "handbags", "test": true}'::jsonb);

-- ============================================================================
-- TEST USERS
-- ============================================================================

-- Test Consumer 1
INSERT INTO users (user_id, external_id, email, status, metadata) VALUES
    ('77777777-7777-7777-7777-777777777777',
     'test_user_001',
     'test.consumer1@example.com',
     'Active',
     '{"test": true, "role": "consumer"}'::jsonb);

-- Test Consumer 2
INSERT INTO users (user_id, external_id, email, status, metadata) VALUES
    ('88888888-8888-8888-8888-888888888888',
     'test_user_002',
     'test.consumer2@example.com',
     'Active',
     '{"test": true, "role": "consumer"}'::jsonb);

-- Test Factory Operator
INSERT INTO users (user_id, external_id, email, status, metadata) VALUES
    ('99999999-9999-9999-9999-999999999999',
     'test_factory_001',
     'factory.operator@example.com',
     'Active',
     '{"test": true, "role": "factory_operator"}'::jsonb);

-- ============================================================================
-- TEST ASSETS (PreMinted and FactoryLogged states)
-- ============================================================================

-- Asset 1: PreMinted watch (not yet logged by factory)
INSERT INTO assets (asset_id, uid, brand_id, current_state, key_epoch, metadata) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
     '04E1A2B3C4D5E6F7',
     'brand_luxury_watch',
     'PreMinted',
     0,
     '{"test": true, "product_type": "luxury_watch"}'::jsonb);

-- Asset 2: FactoryLogged watch (logged but not activated)
INSERT INTO assets (asset_id, uid, brand_id, batch_id, current_state, previous_state, key_epoch, metadata) VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
     '04F1A2B3C4D5E6F8',
     'brand_luxury_watch',
     '55555555-5555-5555-5555-555555555555',
     'FactoryLogged',
     'PreMinted',
     0,
     '{"test": true, "product_type": "luxury_watch", "serial": "LW-2026-001"}'::jsonb);

-- Asset 3: Unassigned watch (logged and ready for activation)
INSERT INTO assets (asset_id, uid, brand_id, batch_id, current_state, previous_state, key_epoch, metadata) VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccccc',
     '04F1A2B3C4D5E6F9',
     'brand_luxury_watch',
     '55555555-5555-5555-5555-555555555555',
     'Unassigned',
     'FactoryLogged',
     0,
     '{"test": true, "product_type": "luxury_watch", "serial": "LW-2026-002"}'::jsonb);

-- Asset 4: Activated watch with external SKU mapping
INSERT INTO assets (
    asset_id, uid, brand_id, batch_id,
    external_product_id, external_product_name, external_product_url,
    current_state, previous_state, key_epoch, activated_at, metadata
) VALUES
    ('dddddddd-dddd-dddd-dddd-dddddddddddd',
     '04F1A2B3C4D5E6FA',
     'brand_luxury_watch',
     '55555555-5555-5555-5555-555555555555',
     'SKU-LW-CHRONO-2026',
     'Chronograph Master Edition 2026',
     'https://luxurywatch.example.com/products/chrono-master-2026',
     'Activated',
     'EntangledPending',
     1,
     NOW() - INTERVAL '5 days',
     '{"test": true, "product_type": "luxury_watch", "serial": "LW-2026-003"}'::jsonb);

-- Asset 5: LegallySold watch with owner
INSERT INTO assets (
    asset_id, uid, brand_id, batch_id,
    external_product_id, external_product_name,
    current_state, previous_state, owner_id, key_epoch, activated_at, sold_at, metadata
) VALUES
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     '04F1A2B3C4D5E6FB',
     'brand_luxury_watch',
     '55555555-5555-5555-5555-555555555555',
     'SKU-LW-CHRONO-2026',
     'Chronograph Master Edition 2026',
     'LegallySold',
     'Activated',
     '77777777-7777-7777-7777-777777777777',
     1,
     NOW() - INTERVAL '10 days',
     NOW() - INTERVAL '3 days',
     '{"test": true, "product_type": "luxury_watch", "serial": "LW-2026-004"}'::jsonb);

-- Asset 6: Fashion handbag (Unassigned)
INSERT INTO assets (asset_id, uid, brand_id, batch_id, current_state, previous_state, key_epoch, metadata) VALUES
    ('ffffffff-ffff-ffff-ffff-ffffffffffff',
     '04F1A2B3C4D5E6FC',
     'brand_fashion',
     '66666666-6666-6666-6666-666666666666',
     'Unassigned',
     'FactoryLogged',
     0,
     '{"test": true, "product_type": "handbag", "collection": "Spring 2026"}'::jsonb);

-- ============================================================================
-- TEST AUTHORITY DEVICES (Virtual Mother Cards)
-- ============================================================================

-- Virtual mother card for Asset 4 (Activated watch)
INSERT INTO authority_devices (
    device_id, brand_id, authority_type,
    virtual_credential_hash, encrypted_k_chip_mother,
    key_derivation_params, owner_id, status, metadata
) VALUES
    ('10000000-0000-0000-0000-000000000001',
     'brand_luxury_watch',
     'VIRTUAL_APP',
     'virtual_token_hash_asset_dddd',
     'encrypted_key_data_placeholder',
     '{"uid": "04F1A2B3C4D5E6FA", "epoch": 1}'::jsonb,
     NULL, -- Not yet assigned to user
     'Active',
     '{"test": true, "auto_generated": true}'::jsonb);

-- Virtual mother card for Asset 5 (LegallySold watch, owned by user)
INSERT INTO authority_devices (
    device_id, brand_id, authority_type,
    virtual_credential_hash, encrypted_k_chip_mother,
    key_derivation_params, owner_id, status, metadata
) VALUES
    ('10000000-0000-0000-0000-000000000002',
     'brand_luxury_watch',
     'VIRTUAL_APP',
     'virtual_token_hash_asset_eeee',
     'encrypted_key_data_placeholder',
     '{"uid": "04F1A2B3C4D5E6FB", "epoch": 1}'::jsonb,
     '77777777-7777-7777-7777-777777777777',
     'Active',
     '{"test": true, "auto_generated": true}'::jsonb);

-- Physical NFC mother card (for high-value testing)
INSERT INTO authority_devices (
    device_id, brand_id, authority_type,
    physical_chip_uid, last_known_ctr, status, metadata
) VALUES
    ('10000000-0000-0000-0000-000000000003',
     'brand_luxury_watch',
     'PHYSICAL_NFC',
     '04A1B2C3D4E5F601',
     0,
     'Active',
     '{"test": true, "physical_card": true}'::jsonb);

-- ============================================================================
-- TEST ENTANGLEMENTS (Mother-Child Bindings)
-- ============================================================================

-- Entanglement for Asset 4
INSERT INTO asset_entanglements (
    entanglement_id, asset_id, authority_device_id, status, created_by, metadata
) VALUES
    ('20000000-0000-0000-0000-000000000001',
     'dddddddd-dddd-dddd-dddd-dddddddddddd',
     '10000000-0000-0000-0000-000000000001',
     'Active',
     'system',
     '{"test": true, "binding_type": "virtual"}'::jsonb);

-- Entanglement for Asset 5
INSERT INTO asset_entanglements (
    entanglement_id, asset_id, authority_device_id, status, created_by, metadata
) VALUES
    ('20000000-0000-0000-0000-000000000002',
     'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     '10000000-0000-0000-0000-000000000002',
     'Active',
     'system',
     '{"test": true, "binding_type": "virtual"}'::jsonb);

-- ============================================================================
-- TEST STATE EVENTS (Audit Trail)
-- ============================================================================

-- Asset 2: PreMinted -> FactoryLogged
INSERT INTO asset_state_events (
    event_id, asset_id, from_state, to_state, action,
    actor_id, actor_role, trace_id, occurred_at, metadata
) VALUES
    ('30000000-0000-0000-0000-000000000001',
     'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
     'PreMinted', 'FactoryLogged', 'BlindLog',
     'factory_swiss_001', 'Factory',
     'trace_test_001',
     NOW() - INTERVAL '15 days',
     '{"test": true, "batch_id": "55555555-5555-5555-5555-555555555555"}'::jsonb);

-- Asset 3: FactoryLogged -> Unassigned
INSERT INTO asset_state_events (
    event_id, asset_id, from_state, to_state, action,
    actor_id, actor_role, trace_id, occurred_at, metadata
) VALUES
    ('30000000-0000-0000-0000-000000000002',
     'cccccccc-cccc-cccc-cccc-cccccccccccc',
     'FactoryLogged', 'Unassigned', 'StockIn',
     'brand_luxury_watch', 'Brand',
     'trace_test_002',
     NOW() - INTERVAL '12 days',
     '{"test": true}'::jsonb);

-- Asset 4: Unassigned -> Activated
INSERT INTO asset_state_events (
    event_id, asset_id, from_state, to_state, action,
    actor_id, actor_role, trace_id, occurred_at, metadata
) VALUES
    ('30000000-0000-0000-0000-000000000003',
     'dddddddd-dddd-dddd-dddd-dddddddddddd',
     'EntangledPending', 'Activated', 'ActivateConfirm',
     'brand_luxury_watch', 'Brand',
     'trace_test_003',
     NOW() - INTERVAL '5 days',
     '{"test": true, "external_product_id": "SKU-LW-CHRONO-2026"}'::jsonb);

-- Asset 5: Activated -> LegallySold
INSERT INTO asset_state_events (
    event_id, asset_id, from_state, to_state, action,
    actor_id, actor_role, trace_id, occurred_at, metadata
) VALUES
    ('30000000-0000-0000-0000-000000000004',
     'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     'Activated', 'LegallySold', 'LegalSell',
     'brand_luxury_watch', 'Brand',
     'trace_test_004',
     NOW() - INTERVAL '3 days',
     '{"test": true, "buyer_id": "77777777-7777-7777-7777-777777777777"}'::jsonb);

-- ============================================================================
-- TEST VERIFICATION LOGS
-- ============================================================================

-- Successful verification of Asset 5
INSERT INTO verification_logs (
    log_id, asset_id, uid, auth_result, cmac_valid, ctr_value, ctr_anomaly,
    ip_address, verified_at, metadata
) VALUES
    ('40000000-0000-0000-0000-000000000001',
     'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     '04F1A2B3C4D5E6FB',
     'Valid',
     TRUE,
     5,
     FALSE,
     '192.168.1.100',
     NOW() - INTERVAL '1 day',
     '{"test": true, "scan_location": "retail_store"}'::jsonb);

-- Suspicious verification attempt
INSERT INTO verification_logs (
    log_id, asset_id, uid, auth_result, cmac_valid, ctr_value, ctr_anomaly,
    ip_address, verified_at, metadata
) VALUES
    ('40000000-0000-0000-0000-000000000002',
     'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
     '04F1A2B3C4D5E6FB',
     'Suspicious',
     FALSE,
     3,
     TRUE,
     '10.0.0.50',
     NOW() - INTERVAL '2 hours',
     '{"test": true, "reason": "ctr_rollback_detected"}'::jsonb);

-- ============================================================================
-- SUMMARY
-- ============================================================================

-- Display seed data summary
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'RCProtocol Seed Data Loaded Successfully';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Brands: %', (SELECT COUNT(*) FROM brands WHERE metadata->>'test' = 'true');
    RAISE NOTICE 'API Keys: %', (SELECT COUNT(*) FROM api_keys);
    RAISE NOTICE 'Webhooks: %', (SELECT COUNT(*) FROM webhook_configs);
    RAISE NOTICE 'Batches: %', (SELECT COUNT(*) FROM batches);
    RAISE NOTICE 'Assets: %', (SELECT COUNT(*) FROM assets);
    RAISE NOTICE 'Authority Devices: %', (SELECT COUNT(*) FROM authority_devices);
    RAISE NOTICE 'Entanglements: %', (SELECT COUNT(*) FROM asset_entanglements);
    RAISE NOTICE 'State Events: %', (SELECT COUNT(*) FROM asset_state_events);
    RAISE NOTICE 'Users: %', (SELECT COUNT(*) FROM users WHERE metadata->>'test' = 'true');
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Test API Keys:';
    RAISE NOTICE '  Luxury Watch: rc_test_luxurywatch_abc123xyz';
    RAISE NOTICE '  Fashion: rc_test_fashion_def456uvw';
    RAISE NOTICE '========================================';
END $$;
