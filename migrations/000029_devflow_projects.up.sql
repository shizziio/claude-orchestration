-- DevFlow Extension: Projects + Git Credentials + Environments + Runs

-- ============================================================
-- ext_git_credentials: SSH keys / tokens for git providers
-- ============================================================

CREATE TABLE ext_git_credentials (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      VARCHAR(255) NOT NULL,
    label        VARCHAR(255) NOT NULL,
    provider     VARCHAR(50)  NOT NULL DEFAULT 'custom', -- github, gitlab, bitbucket, custom
    host         VARCHAR(255),                           -- e.g. gitlab.mycompany.com
    auth_type    VARCHAR(20)  NOT NULL DEFAULT 'ssh_key', -- ssh_key | token
    public_key   TEXT,                                   -- SSH public key (display to user)
    private_key  BYTEA,                                  -- AES-256-GCM encrypted SSH private key
    token        BYTEA,                                  -- AES-256-GCM encrypted PAT/token
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_git_creds_tenant      ON ext_git_credentials(tenant_id);
CREATE INDEX idx_ext_git_creds_tenant_user ON ext_git_credentials(tenant_id, user_id);

-- ============================================================
-- ext_projects: software projects managed by DevFlow
-- ============================================================

CREATE TABLE ext_projects (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    slug                 VARCHAR(100) NOT NULL,
    description          TEXT,
    repo_url             TEXT,                           -- git clone URL
    default_branch       VARCHAR(255) NOT NULL DEFAULT 'main',
    git_credential_id    UUID REFERENCES ext_git_credentials(id) ON DELETE SET NULL,
    workspace_path       TEXT,                           -- absolute local path (cloned repo)
    docker_compose_file  TEXT,                           -- relative path inside repo, nullable
    status               VARCHAR(20) NOT NULL DEFAULT 'active', -- active | archived
    settings             JSONB NOT NULL DEFAULT '{}',
    created_by           VARCHAR(255) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_ext_projects_tenant_slug ON ext_projects(tenant_id, slug);
CREATE INDEX idx_ext_projects_tenant            ON ext_projects(tenant_id);
CREATE INDEX idx_ext_projects_tenant_active     ON ext_projects(tenant_id) WHERE status = 'active';

-- ============================================================
-- ext_environments: per-project environments (dev, staging…)
-- ============================================================

CREATE TABLE ext_environments (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id               UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    name                     VARCHAR(100) NOT NULL,      -- dev, staging, prod, or custom
    slug                     VARCHAR(100) NOT NULL,
    status                   VARCHAR(20) NOT NULL DEFAULT 'dormant', -- dormant | coding | running
    branch                   VARCHAR(255),               -- override branch for this env
    env_vars                 BYTEA,                      -- AES-256-GCM encrypted KEY=VAL lines
    docker_compose_override  TEXT,                       -- relative path to override file
    port_bindings            JSONB NOT NULL DEFAULT '{}', -- {"3000": 3001, ...}
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_ext_environments_project_slug ON ext_environments(project_id, slug);
CREATE INDEX idx_ext_environments_tenant              ON ext_environments(tenant_id);
CREATE INDEX idx_ext_environments_project             ON ext_environments(project_id);
CREATE INDEX idx_ext_environments_status              ON ext_environments(status) WHERE status != 'dormant';

-- ============================================================
-- ext_devflow_runs: Claude Code agent run tracking
-- ============================================================

CREATE TABLE ext_devflow_runs (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id        UUID NOT NULL REFERENCES ext_projects(id) ON DELETE CASCADE,
    environment_id    UUID REFERENCES ext_environments(id) ON DELETE SET NULL,
    task_description  TEXT NOT NULL,
    context_prompt    TEXT,                              -- tier-2 injected docs (ExtraSystemPrompt)
    branch            VARCHAR(255),                     -- git branch for this run
    status            VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | running | completed | failed
    claude_session_id VARCHAR(255),                     -- Claude Code session ID
    result_summary    TEXT,
    error_message     TEXT,
    created_by        VARCHAR(255) NOT NULL,
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ext_devflow_runs_tenant         ON ext_devflow_runs(tenant_id);
CREATE INDEX idx_ext_devflow_runs_project        ON ext_devflow_runs(project_id);
CREATE INDEX idx_ext_devflow_runs_status         ON ext_devflow_runs(tenant_id, status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_ext_devflow_runs_created        ON ext_devflow_runs(tenant_id, created_at DESC);
