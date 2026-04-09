CREATE TABLE IF NOT EXISTS workorders (
    id                UUID PRIMARY KEY,
    type              VARCHAR(50)  NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'open',
    title             TEXT         NOT NULL,
    description       TEXT,
    creator_id        UUID         NOT NULL,
    creator_role      VARCHAR(50)  NOT NULL,
    creator_org_id    UUID         NOT NULL,
    assignee_id       UUID,
    assignee_role     VARCHAR(50),
    asset_id          VARCHAR(255),
    brand_id          VARCHAR(255),
    conclusion        TEXT,
    conclusion_type   VARCHAR(50),
    approval_id       UUID,
    downstream_result JSONB,
    metadata          JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workorders_type_status ON workorders(type, status);
CREATE INDEX IF NOT EXISTS idx_workorders_creator_id ON workorders(creator_id);
CREATE INDEX IF NOT EXISTS idx_workorders_assignee_id ON workorders(assignee_id);
CREATE INDEX IF NOT EXISTS idx_workorders_asset_id ON workorders(asset_id);
