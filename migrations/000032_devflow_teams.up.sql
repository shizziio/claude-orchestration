CREATE TABLE ext_project_teams (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    team_name    VARCHAR(100) NOT NULL,
    description  TEXT,
    team_config  JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_project_teams_tenant  ON ext_project_teams(tenant_id);
CREATE INDEX idx_ext_project_teams_project ON ext_project_teams(project_id);
CREATE UNIQUE INDEX idx_ext_project_teams_name ON ext_project_teams(project_id, team_name);
