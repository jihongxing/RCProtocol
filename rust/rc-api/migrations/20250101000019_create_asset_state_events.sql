-- Add missing columns to asset_state_events table
ALTER TABLE asset_state_events ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE asset_state_events ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

-- Create missing indexes
CREATE INDEX IF NOT EXISTS idx_state_events_occurred_at ON asset_state_events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_state_events_action ON asset_state_events(action);
CREATE INDEX IF NOT EXISTS idx_state_events_actor_id ON asset_state_events(actor_id);
