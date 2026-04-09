-- Spec-10: Create approvals table for go-approval service (FR-03)
CREATE TABLE IF NOT EXISTS approvals (
    id                UUID PRIMARY KEY,
    type              VARCHAR(50)  NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'pending',
    applicant_id      UUID         NOT NULL,
    applicant_role    VARCHAR(50)  NOT NULL,
    applicant_org_id  UUID         NOT NULL,
    reviewer_id       UUID,
    reviewer_role     VARCHAR(50),
    payload           JSONB        NOT NULL,
    reason            TEXT,
    review_comment    TEXT,
    resource_type     VARCHAR(50)  NOT NULL,
    resource_id       VARCHAR(255) NOT NULL,
    downstream_result JSONB,
    expires_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_approvals_type_status ON approvals(type, status);
CREATE INDEX IF NOT EXISTS idx_approvals_applicant_id ON approvals(applicant_id);
CREATE INDEX IF NOT EXISTS idx_approvals_resource ON approvals(resource_type, resource_id);
