CREATE TABLE IF NOT EXISTS asset_state_events (
    event_id UUID PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    action TEXT NOT NULL,
    from_state TEXT NULL,
    to_state TEXT NOT NULL,
    trace_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    actor_org TEXT NULL,
    idempotency_key TEXT NOT NULL,
    approval_id TEXT NULL,
    policy_version TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
