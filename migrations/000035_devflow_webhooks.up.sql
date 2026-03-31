CREATE TABLE ext_devflow_webhooks (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    event_type   VARCHAR(50) NOT NULL DEFAULT 'push',  -- push | pull_request
    branch_filter VARCHAR(255),                         -- regex or glob, NULL = all branches
    task_template TEXT NOT NULL,                         -- task prompt template, supports {{.Branch}}, {{.CommitMsg}}
    enabled      BOOLEAN NOT NULL DEFAULT true,
    secret       VARCHAR(255),                          -- webhook secret for signature validation
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_devflow_webhooks_tenant  ON ext_devflow_webhooks(tenant_id);
CREATE INDEX idx_ext_devflow_webhooks_project ON ext_devflow_webhooks(project_id);
