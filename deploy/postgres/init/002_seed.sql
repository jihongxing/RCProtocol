-- RCProtocol local development seed data
-- Purpose: provide deterministic assets for end-to-end API flow testing.

INSERT INTO brands (brand_id, brand_name)
VALUES ('brand-demo', 'RC Demo Brand')
ON CONFLICT (brand_id) DO NOTHING;

INSERT INTO products (product_id, brand_id, product_name)
VALUES ('product-demo-001', 'brand-demo', 'RC Demo Product')
ON CONFLICT (product_id) DO NOTHING;

INSERT INTO batches (batch_id, brand_id)
VALUES ('batch-demo-001', 'brand-demo')
ON CONFLICT (batch_id) DO NOTHING;

INSERT INTO factory_sessions (session_id, batch_id)
VALUES ('session-demo-001', 'batch-demo-001')
ON CONFLICT (session_id) DO NOTHING;

INSERT INTO assets (asset_id, brand_id, product_id, uid, current_state, previous_state)
VALUES
    ('asset-main-001', 'brand-demo', 'product-demo-001', 'UID-DEMO-0001', 'PreMinted', NULL),
    ('asset-freeze-001', 'brand-demo', 'product-demo-001', 'UID-DEMO-0002', 'Activated', NULL),
    ('asset-transfer-001', 'brand-demo', 'product-demo-001', 'UID-DEMO-0003', 'LegallySold', NULL),
    ('asset-terminal-001', 'brand-demo', 'product-demo-001', 'UID-DEMO-0004', 'Transferred', NULL)
ON CONFLICT (asset_id) DO NOTHING;
