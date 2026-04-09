-- Create approval_records table for Platform approval workflow
CREATE TABLE IF NOT EXISTS approval_records (
    approval_id TEXT PRIMARY KEY,
    requester_id TEXT NOT NULL,
    approver_id TEXT,
    operation_type TEXT NOT NULL,
    target_resource TEXT NOT NULL,
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'Pending',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT valid_status CHECK (status IN ('Pending', 'Approved', 'Rejected', 'Expired', 'Used'))
);

-- Indexes for efficient queries
CREATE INDEX idx_approval_status ON approval_records(status, expires_at);
CREATE INDEX idx_approval_requester ON approval_records(requester_id);
CREATE INDEX idx_approval_created ON approval_records(created_at DESC);

-- Comments
COMMENT ON TABLE approval_records IS 'Platform approval workflow records';
COMMENT ON COLUMN approval_records.approval_id IS 'Unique approval identifier (appr_xxx format)';
COMMENT ON COLUMN approval_records.requester_id IS 'User who requested the approval';
COMMENT ON COLUMN approval_records.approver_id IS 'User who approved/rejected the request';
COMMENT ON COLUMN approval_records.operation_type IS 'Type of operation (StockIn, ActivateRotateKeys, etc)';
COMMENT ON COLUMN approval_records.target_resource IS 'Target asset_id or batch_id';
COMMENT ON COLUMN approval_records.status IS 'Pending, Approved, Rejected, Expired, or Used';
