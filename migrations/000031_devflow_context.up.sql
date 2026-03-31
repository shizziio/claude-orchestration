-- ============================================================
-- ext_project_context: Tier 1 context entries → feed CLAUDE.md
-- ============================================================
CREATE TABLE ext_project_context (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    doc_type     VARCHAR(30) NOT NULL DEFAULT 'rules',  -- rules | index | structure
    title        VARCHAR(255) NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_project_context_tenant   ON ext_project_context(tenant_id);
CREATE INDEX idx_ext_project_context_project  ON ext_project_context(project_id, doc_type, sort_order);
CREATE UNIQUE INDEX idx_ext_project_context_title ON ext_project_context(project_id, title);

-- ============================================================
-- ext_task_context: Tier 2 document pool (story specs, designs…)
-- ============================================================
CREATE TABLE ext_task_context (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    tags         TEXT[] DEFAULT '{}',
    file_path    VARCHAR(500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_task_context_tenant   ON ext_task_context(tenant_id);
CREATE INDEX idx_ext_task_context_project  ON ext_task_context(project_id);
CREATE INDEX idx_ext_task_context_tags     ON ext_task_context USING GIN(tags);

-- ============================================================
-- ext_task_context_refs: run ↔ document attachments
-- ============================================================
CREATE TABLE ext_task_context_refs (
    id               UUID PRIMARY KEY,
    run_id           UUID NOT NULL REFERENCES ext_devflow_runs(id) ON DELETE CASCADE,
    task_context_id  UUID NOT NULL REFERENCES ext_task_context(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_ext_task_context_refs_unique ON ext_task_context_refs(run_id, task_context_id);
CREATE INDEX idx_ext_task_context_refs_run           ON ext_task_context_refs(run_id);
