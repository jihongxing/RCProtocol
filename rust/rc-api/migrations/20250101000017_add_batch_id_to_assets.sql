-- Add batch_id column to assets table
ALTER TABLE assets ADD COLUMN batch_id TEXT NULL REFERENCES batches(batch_id);

-- Create index for batch_id lookups
CREATE INDEX idx_assets_batch_id ON assets(batch_id);
