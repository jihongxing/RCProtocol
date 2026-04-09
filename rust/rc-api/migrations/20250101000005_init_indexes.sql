CREATE INDEX IF NOT EXISTS idx_assets_brand_id ON assets(brand_id);
CREATE INDEX IF NOT EXISTS idx_assets_current_state ON assets(current_state);
CREATE INDEX IF NOT EXISTS idx_asset_state_events_asset_id ON asset_state_events(asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_state_events_trace_id ON asset_state_events(trace_id);
