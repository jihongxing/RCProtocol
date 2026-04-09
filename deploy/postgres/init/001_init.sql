-- RCProtocol Database Schema
-- Version: 1.0
-- Last Updated: 2026-04-08
-- Description: Core protocol tables for asset lifecycle management

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- BRANDS TABLE
-- ============================================================================
CREATE TABLE brands (
    brand_id VARCHAR(64) PRIMARY KEY,
    brand_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL UNIQUE, -- Contact email for brand (unique)
    industry VARCHAR(50) NOT NULL, -- Industry type: Watches, Fashion, Wine, Jewelry, Art, Other
    brand_key_hash VARCHAR(128), -- HMAC-SHA256 hash for verification (optional for MVP)
    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Suspended, Archived
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_brands_status ON brands(status);
CREATE INDEX idx_brands_created_at ON brands(created_at);
CREATE UNIQUE INDEX idx_brands_contact_email ON brands(contact_email);

-- ============================================================================
-- API KEYS TABLE
-- ============================================================================
CREATE TABLE api_keys (
    key_id VARCHAR(64) PRIMARY KEY, -- ULID format: key_01HQZX...
    brand_id VARCHAR(64) NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,
    key_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of the API key (64 hex chars)
    key_prefix VARCHAR(20) NOT NULL, -- First 16 chars for display (e.g., "rcpk_live_1234****")
    name VARCHAR(255), -- Optional friendly name
    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Revoked
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64),
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(64),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_api_keys_brand_id ON api_keys(brand_id);
CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- ============================================================================
-- WEBHOOK CONFIGS TABLE
-- ============================================================================
CREATE TABLE webhook_configs (
    webhook_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    brand_id VARCHAR(64) NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret VARCHAR(128), -- HMAC secret for signature verification
    events TEXT[] NOT NULL, -- Array of event types: ['asset.activated', 'asset.transferred', etc.]
    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Disabled
    retry_config JSONB DEFAULT '{"max_retries": 3, "backoff_seconds": [60, 300, 900]}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_configs_brand_id ON webhook_configs(brand_id);
CREATE INDEX idx_webhook_configs_status ON webhook_configs(status);

-- ============================================================================
-- WEBHOOK DELIVERIES TABLE
-- ============================================================================
CREATE TABLE webhook_deliveries (
    delivery_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID NOT NULL REFERENCES webhook_configs(webhook_id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'Pending', -- Pending, Sent, Failed, Abandoned
    attempts INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,
    response_status_code INT,
    response_body TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX idx_webhook_deliveries_next_retry ON webhook_deliveries(next_retry_at) WHERE status = 'Pending';

-- ============================================================================
-- BATCHES TABLE (for factory blind scan sessions)
-- ============================================================================
CREATE TABLE batches (
    batch_id TEXT PRIMARY KEY,
    brand_id VARCHAR(64) NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,
    batch_name VARCHAR(255),
    factory_id VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'Open', -- Open, Closed, Cancelled
    expected_count INT,
    actual_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_batches_brand_id ON batches(brand_id);
CREATE INDEX idx_batches_status ON batches(status);
CREATE INDEX idx_batches_created_at ON batches(created_at);

-- ============================================================================
-- ASSETS TABLE
-- ============================================================================
CREATE TABLE assets (
    asset_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uid VARCHAR(32) NOT NULL UNIQUE, -- NFC chip UID (hex)
    brand_id VARCHAR(64) NOT NULL REFERENCES brands(brand_id) ON DELETE RESTRICT,
    batch_id UUID REFERENCES batches(batch_id) ON DELETE SET NULL,

    -- External SKU mapping (not managing full product details)
    external_product_id VARCHAR(255), -- Brand's SKU ID
    external_product_name VARCHAR(512), -- Optional, for display
    external_product_url TEXT, -- Optional, for linking to brand's system

    -- State machine
    current_state VARCHAR(32) NOT NULL DEFAULT 'PreMinted',
    previous_state VARCHAR(32),

    -- Ownership
    owner_id VARCHAR(64), -- Current legal holder (user_id or consumer_id)

    -- Key rotation tracking
    key_epoch INT NOT NULL DEFAULT 0,
    last_key_rotation_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    sold_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT chk_asset_state CHECK (current_state IN (
        'PreMinted', 'FactoryLogged', 'Unassigned', 'RotatingKeys',
        'EntangledPending', 'Activated', 'LegallySold', 'Transferred',
        'Consumed', 'Legacy', 'Tampered', 'Compromised', 'Destructed', 'Disputed'
    ))
);

CREATE INDEX idx_assets_uid ON assets(uid);
CREATE INDEX idx_assets_brand_id ON assets(brand_id);
CREATE INDEX idx_assets_batch_id ON assets(batch_id);
CREATE INDEX idx_assets_current_state ON assets(current_state);
CREATE INDEX idx_assets_owner_id ON assets(owner_id);
CREATE INDEX idx_assets_external_product_id ON assets(external_product_id);
CREATE INDEX idx_assets_created_at ON assets(created_at);

-- ============================================================================
-- AUTHORITY DEVICES TABLE (Mother Tags / Virtual Cards)
-- ============================================================================
CREATE TABLE authority_devices (
    authority_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    brand_id VARCHAR(64) NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,

    -- Authority UID (unique identifier for both physical and virtual)
    authority_uid TEXT NOT NULL UNIQUE,

    -- Device type: PHYSICAL_NFC, VIRTUAL_QR, VIRTUAL_APP, VIRTUAL_BIOMETRIC
    authority_type VARCHAR(32) NOT NULL,

    -- Physical NFC fields
    physical_chip_uid VARCHAR(32), -- Only for PHYSICAL_NFC
    last_known_ctr INT DEFAULT 0,

    -- Virtual card fields
    virtual_credential_hash VARCHAR(128), -- Token hash for virtual cards
    bound_user_id VARCHAR(64), -- User who owns this virtual authority device

    -- Key management
    key_epoch INTEGER NOT NULL DEFAULT 0,

    -- Status
    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Suspended, Revoked, Replaced

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,

    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT chk_authority_type CHECK (authority_type IN (
        'PHYSICAL_NFC', 'VIRTUAL_QR', 'VIRTUAL_APP', 'VIRTUAL_BIOMETRIC'
    )),
    CONSTRAINT chk_authority_status CHECK (status IN (
        'Active', 'Suspended', 'Revoked', 'Replaced'
    ))
);
        'PHYSICAL_NFC', 'VIRTUAL_QR', 'VIRTUAL_APP', 'VIRTUAL_BIOMETRIC'
    )),
    CONSTRAINT chk_physical_uid CHECK (
        (authority_type = 'PHYSICAL_NFC' AND physical_chip_uid IS NOT NULL) OR
        (authority_type != 'PHYSICAL_NFC')
    )
);

CREATE INDEX idx_authority_devices_brand_id ON authority_devices(brand_id);
CREATE INDEX idx_authority_devices_owner_id ON authority_devices(owner_id);
CREATE INDEX idx_authority_devices_physical_uid ON authority_devices(physical_chip_uid);
CREATE INDEX idx_authority_devices_status ON authority_devices(status);
CREATE INDEX idx_authority_devices_type ON authority_devices(authority_type);

-- ============================================================================
-- ASSET ENTANGLEMENTS TABLE (Mother-Child Bindings)
-- ============================================================================
CREATE TABLE asset_entanglements (
    entanglement_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    authority_device_id UUID NOT NULL REFERENCES authority_devices(device_id) ON DELETE CASCADE,

    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Revoked

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64),
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(64),

    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT uq_asset_authority UNIQUE (asset_id, authority_device_id)
);

CREATE INDEX idx_entanglements_asset_id ON asset_entanglements(asset_id);
CREATE INDEX idx_entanglements_authority_id ON asset_entanglements(authority_device_id);
CREATE INDEX idx_entanglements_status ON asset_entanglements(status);

-- ============================================================================
-- ASSET STATE EVENTS TABLE (Audit Trail)
-- ============================================================================
CREATE TABLE asset_state_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,

    -- State transition
    from_state VARCHAR(32),
    to_state VARCHAR(32) NOT NULL,
    action VARCHAR(64) NOT NULL, -- BlindLog, ActivateConfirm, LegalSell, Transfer, etc.

    -- Actor context
    actor_id VARCHAR(64) NOT NULL,
    actor_role VARCHAR(32) NOT NULL, -- Platform, Factory, Brand, Consumer, Moderator

    -- Request context
    trace_id VARCHAR(64),
    idempotency_key VARCHAR(128),

    -- Timestamps
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Additional context
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT chk_event_action CHECK (action IN (
        'BlindLog', 'StockIn', 'ActivateRotateKeys', 'ActivateEntangle',
        'ActivateConfirm', 'LegalSell', 'Transfer', 'Consume', 'Legacy',
        'Freeze', 'Recover', 'MarkTampered', 'MarkCompromised', 'MarkDestructed'
    ))
);

CREATE INDEX idx_state_events_asset_id ON asset_state_events(asset_id);
CREATE INDEX idx_state_events_occurred_at ON asset_state_events(occurred_at);
CREATE INDEX idx_state_events_action ON asset_state_events(action);
CREATE INDEX idx_state_events_actor_id ON asset_state_events(actor_id);
CREATE INDEX idx_state_events_trace_id ON asset_state_events(trace_id);

-- ============================================================================
-- IDEMPOTENCY RECORDS TABLE
-- ============================================================================
CREATE TABLE idempotency_records (
    idempotency_key VARCHAR(128) PRIMARY KEY,
    asset_id UUID REFERENCES assets(asset_id) ON DELETE CASCADE,
    action VARCHAR(64) NOT NULL,
    request_hash VARCHAR(128) NOT NULL,
    response_status INT NOT NULL,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX idx_idempotency_asset_id ON idempotency_records(asset_id);
CREATE INDEX idx_idempotency_expires_at ON idempotency_records(expires_at);

-- ============================================================================
-- VERIFICATION LOGS TABLE (C-side scan tracking)
-- ============================================================================
CREATE TABLE verification_logs (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID REFERENCES assets(asset_id) ON DELETE SET NULL,
    uid VARCHAR(32) NOT NULL,

    -- Authentication result
    auth_result VARCHAR(32) NOT NULL, -- Valid, Invalid, Suspicious
    cmac_valid BOOLEAN,
    ctr_value INT,
    ctr_anomaly BOOLEAN DEFAULT FALSE,

    -- Request context
    ip_address INET,
    user_agent TEXT,
    geo_location JSONB, -- {country, city, lat, lon}

    -- Timestamps
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_verification_logs_asset_id ON verification_logs(asset_id);
CREATE INDEX idx_verification_logs_uid ON verification_logs(uid);
CREATE INDEX idx_verification_logs_verified_at ON verification_logs(verified_at);
CREATE INDEX idx_verification_logs_auth_result ON verification_logs(auth_result);

-- ============================================================================
-- USERS TABLE (Simplified for MVP)
-- ============================================================================
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(128) UNIQUE, -- External identity provider ID
    email VARCHAR(255),
    phone VARCHAR(32),

    -- WebAuthn credentials for biometric auth
    webauthn_credentials JSONB DEFAULT '[]'::jsonb,

    status VARCHAR(32) NOT NULL DEFAULT 'Active', -- Active, Suspended, Deleted

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,

    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- ============================================================================
-- VAULT SNAPSHOTS TABLE (Redis backup)
-- ============================================================================
CREATE TABLE vault_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    asset_ids UUID[] NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vault_snapshots_user_id ON vault_snapshots(user_id);
CREATE INDEX idx_vault_snapshots_created_at ON vault_snapshots(created_at);

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_brands_updated_at BEFORE UPDATE ON brands
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_authority_devices_updated_at BEFORE UPDATE ON authority_devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_webhook_configs_updated_at BEFORE UPDATE ON webhook_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Increment batch actual_count when asset is logged
CREATE OR REPLACE FUNCTION increment_batch_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.batch_id IS NOT NULL THEN
        UPDATE batches
        SET actual_count = actual_count + 1
        WHERE batch_id = NEW.batch_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER increment_batch_count_on_insert AFTER INSERT ON assets
    FOR EACH ROW EXECUTE FUNCTION increment_batch_count();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE brands IS 'Brand registry with minimal 3-field design';
COMMENT ON TABLE api_keys IS 'API keys for brand integration';
COMMENT ON TABLE webhook_configs IS 'Webhook configurations for event notifications';
COMMENT ON TABLE assets IS 'Core asset records with external SKU mapping';
COMMENT ON TABLE authority_devices IS 'Mother tags (physical NFC or virtual cards)';
COMMENT ON TABLE asset_entanglements IS 'Mother-child authorization bindings';
COMMENT ON TABLE asset_state_events IS 'Immutable audit trail of state transitions';
COMMENT ON TABLE idempotency_records IS 'Idempotency protection for write operations';
COMMENT ON TABLE verification_logs IS 'Consumer-side verification tracking';

-- ============================================================================
-- INITIAL DATA (Optional)
-- ============================================================================

-- Create system platform brand
INSERT INTO brands (brand_id, brand_name, contact_email, industry, brand_key_hash, status, metadata) VALUES
    ('platform', 'RCProtocol Platform', 'system@rcprotocol.internal', 'platform', 'system_internal_hash', 'Active', '{"system": true}'::jsonb)
ON CONFLICT (brand_id) DO NOTHING;
