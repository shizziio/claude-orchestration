package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ------------------------------------------------------------
// Types
// ------------------------------------------------------------

// GitCredential holds authentication for a git provider.
type GitCredential struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	Label      string    `json:"label"`
	Provider   string    `json:"provider"`   // github | gitlab | bitbucket | custom
	Host       *string   `json:"host"`       // custom host, e.g. gitlab.corp.com
	AuthType   string    `json:"auth_type"`  // ssh_key | token
	PublicKey  *string   `json:"public_key"` // SSH public key (display only)
	// PrivateKey and Token are never returned to callers — encryption handled in pg layer.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Project is a software project managed by DevFlow.
type Project struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	Name               string     `json:"name"`
	Slug               string     `json:"slug"`
	Description        *string    `json:"description"`
	RepoURL            *string    `json:"repo_url"`
	DefaultBranch      string     `json:"default_branch"`
	GitCredentialID    *uuid.UUID `json:"git_credential_id"`
	WorkspacePath      *string    `json:"workspace_path"`
	DockerComposeFile  *string    `json:"docker_compose_file"`
	Status             string     `json:"status"` // active | archived
	Settings           []byte     `json:"settings"`
	CreatedBy          string     `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Environment is a per-project environment (dev, staging, prod…).
type Environment struct {
	ID                     uuid.UUID `json:"id"`
	TenantID               uuid.UUID `json:"tenant_id"`
	ProjectID              uuid.UUID `json:"project_id"`
	Name                   string    `json:"name"`
	Slug                   string    `json:"slug"`
	Status                 string    `json:"status"`         // dormant | coding | running
	ComposeStatus          string    `json:"compose_status"` // stopped | starting | running | error
	Branch                 *string   `json:"branch"`
	DockerComposeOverride  *string   `json:"docker_compose_override"`
	PortBindings           []byte    `json:"port_bindings"`
	CodeServerPort         *int      `json:"code_server_port"`
	PreviewPort            *int      `json:"preview_port"`
	// EnvVars is never returned — decryption handled in pg layer.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DevflowRun tracks a single Claude Code agent run.
type DevflowRun struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	EnvironmentID   *uuid.UUID `json:"environment_id"`
	TaskDescription string     `json:"task_description"`
	ContextPrompt   *string    `json:"context_prompt"`
	Branch          *string    `json:"branch"`
	Status          string     `json:"status"` // pending | running | completed | failed
	ClaudeSessionID *string    `json:"claude_session_id"`
	ResultSummary   *string    `json:"result_summary"`
	ErrorMessage    *string    `json:"error_message"`
	RunLog          *string    `json:"run_log"`
	CostUSD         *float64   `json:"cost_usd"`
	DurationMs      *int       `json:"duration_ms"`
	CreatedBy       string     `json:"created_by"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ------------------------------------------------------------
// Create/Update input types
// ------------------------------------------------------------

type CreateGitCredentialInput struct {
	UserID     string
	Label      string
	Provider   string
	Host       *string
	AuthType   string
	PublicKey  *string
	PrivateKey []byte // plaintext; pg layer encrypts
	Token      []byte // plaintext; pg layer encrypts
}

type UpdateGitCredentialInput struct {
	Label      *string
	Host       *string
	PublicKey  *string
	PrivateKey []byte // nil = keep existing; non-nil = replace (pg layer encrypts)
	Token      []byte // nil = keep existing; non-nil = replace (pg layer encrypts)
}

type CreateProjectInput struct {
	Name              string
	Slug              string
	Description       *string
	RepoURL           *string
	DefaultBranch     string
	GitCredentialID   *uuid.UUID
	DockerComposeFile *string
	Settings          []byte
	CreatedBy         string
}

type UpdateProjectInput struct {
	Name              *string
	Description       *string
	RepoURL           *string
	DefaultBranch     *string
	GitCredentialID   *uuid.UUID
	WorkspacePath     *string
	DockerComposeFile *string
	Status            *string
	Settings          []byte
}

type CreateEnvironmentInput struct {
	ProjectID             uuid.UUID
	Name                  string
	Slug                  string
	Branch                *string
	EnvVars               []byte // plaintext; pg layer encrypts
	DockerComposeOverride *string
	PortBindings          []byte
}

type UpdateEnvironmentInput struct {
	Status                *string
	ComposeStatus         *string
	Branch                *string
	EnvVars               []byte
	DockerComposeOverride *string
	PortBindings          []byte
	CodeServerPort        *int
	PreviewPort           *int
}

type CreateRunInput struct {
	ProjectID       uuid.UUID
	EnvironmentID   *uuid.UUID
	TaskDescription string
	ContextPrompt   *string
	Branch          *string
	CreatedBy       string
}

type UpdateRunInput struct {
	Status          *string
	ClaudeSessionID *string
	ResultSummary   *string
	ErrorMessage    *string
	RunLog          *string
	CostUSD         *float64
	DurationMs      *int
	StartedAt       *time.Time
	CompletedAt     *time.Time
}

// ProjectContext is a Tier-1 context entry that feeds CLAUDE.md.
type ProjectContext struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ProjectID uuid.UUID `json:"project_id"`
	DocType   string    `json:"doc_type"` // rules | index | structure
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskContext is a Tier-2 document in the pool (story specs, designs, etc.).
type TaskContext struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ProjectID uuid.UUID `json:"project_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	FilePath  *string   `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskContextRef links a run to a task context document.
type TaskContextRef struct {
	ID            uuid.UUID `json:"id"`
	RunID         uuid.UUID `json:"run_id"`
	TaskContextID uuid.UUID `json:"task_context_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateProjectContextInput struct {
	ProjectID uuid.UUID
	DocType   string
	Title     string
	Content   string
	SortOrder int
}

type UpdateProjectContextInput struct {
	DocType   *string
	Title     *string
	Content   *string
	SortOrder *int
}

type CreateTaskContextInput struct {
	ProjectID uuid.UUID
	Title     string
	Content   string
	Tags      []string
	FilePath  *string
}

type UpdateTaskContextInput struct {
	Title    *string
	Content  *string
	Tags     []string // nil = keep existing; non-nil = replace
	FilePath *string
}

// DevflowWebhook configures auto-run triggers on git events.
type DevflowWebhook struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	ProjectID    uuid.UUID `json:"project_id"`
	EventType    string    `json:"event_type"`    // push | pull_request
	BranchFilter *string   `json:"branch_filter"` // regex/glob, nil = all
	TaskTemplate string    `json:"task_template"`
	Enabled      bool      `json:"enabled"`
	Secret       *string   `json:"secret"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateWebhookInput struct {
	ProjectID    uuid.UUID
	EventType    string
	BranchFilter *string
	TaskTemplate string
	Secret       *string
}

type UpdateWebhookInput struct {
	EventType    *string
	BranchFilter *string
	TaskTemplate *string
	Enabled      *bool
	Secret       *string
}

// ProjectTeam defines an agent team configuration for a project.
type ProjectTeam struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	ProjectID   uuid.UUID       `json:"project_id"`
	TeamName    string          `json:"team_name"`
	Description *string         `json:"description"`
	TeamConfig  json.RawMessage `json:"team_config"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateProjectTeamInput struct {
	ProjectID   uuid.UUID
	TeamName    string
	Description *string
	TeamConfig  json.RawMessage
}

type UpdateProjectTeamInput struct {
	TeamName    *string
	Description *string
	TeamConfig  json.RawMessage // nil = keep existing
}

// ------------------------------------------------------------
// Interfaces
// ------------------------------------------------------------

// GitCredentialStore manages git authentication credentials.
type GitCredentialStore interface {
	Create(ctx context.Context, in CreateGitCredentialInput) (*GitCredential, error)
	Get(ctx context.Context, id uuid.UUID) (*GitCredential, error)
	List(ctx context.Context, userID string) ([]*GitCredential, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateGitCredentialInput) (*GitCredential, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// GetPrivateKey returns the decrypted SSH private key. Never exposed via API.
	GetPrivateKey(ctx context.Context, id uuid.UUID) ([]byte, error)
	// GetToken returns the decrypted PAT/token. Never exposed via API.
	GetToken(ctx context.Context, id uuid.UUID) ([]byte, error)
}

// ProjectStore manages DevFlow projects.
type ProjectStore interface {
	Create(ctx context.Context, in CreateProjectInput) (*Project, error)
	Get(ctx context.Context, id uuid.UUID) (*Project, error)
	GetBySlug(ctx context.Context, slug string) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateProjectInput) (*Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EnvironmentStore manages per-project environments.
type EnvironmentStore interface {
	Create(ctx context.Context, in CreateEnvironmentInput) (*Environment, error)
	Get(ctx context.Context, id uuid.UUID) (*Environment, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*Environment, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateEnvironmentInput) (*Environment, error)
	GetDecryptedEnvVars(ctx context.Context, id uuid.UUID) ([]byte, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DevflowRunStore tracks Claude Code agent runs.
type DevflowRunStore interface {
	Create(ctx context.Context, in CreateRunInput) (*DevflowRun, error)
	Get(ctx context.Context, id uuid.UUID) (*DevflowRun, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]*DevflowRun, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateRunInput) (*DevflowRun, error)
	// GetLog returns the persisted run_log for a completed run. Returns ("", nil) if not set.
	GetLog(ctx context.Context, id uuid.UUID) (string, error)
}

// ProjectContextStore manages Tier-1 context entries that feed CLAUDE.md.
type ProjectContextStore interface {
	Create(ctx context.Context, in CreateProjectContextInput) (*ProjectContext, error)
	Get(ctx context.Context, id uuid.UUID) (*ProjectContext, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, docType string) ([]*ProjectContext, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateProjectContextInput) (*ProjectContext, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// TaskContextStore manages Tier-2 document pool.
type TaskContextStore interface {
	Create(ctx context.Context, in CreateTaskContextInput) (*TaskContext, error)
	Get(ctx context.Context, id uuid.UUID) (*TaskContext, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*TaskContext, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateTaskContextInput) (*TaskContext, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// TaskContextRefStore manages run ↔ document attachments.
type TaskContextRefStore interface {
	Attach(ctx context.Context, runID uuid.UUID, contextIDs []uuid.UUID) error
	ListByRun(ctx context.Context, runID uuid.UUID) ([]*TaskContextRef, error)
	Detach(ctx context.Context, runID, contextID uuid.UUID) error
}

// DevflowWebhookStore manages webhook triggers for auto-runs.
type DevflowWebhookStore interface {
	Create(ctx context.Context, in CreateWebhookInput) (*DevflowWebhook, error)
	Get(ctx context.Context, id uuid.UUID) (*DevflowWebhook, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*DevflowWebhook, error)
	ListEnabled(ctx context.Context, projectID uuid.UUID, eventType string) ([]*DevflowWebhook, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateWebhookInput) (*DevflowWebhook, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectTeamStore manages agent team configurations per project.
type ProjectTeamStore interface {
	Create(ctx context.Context, in CreateProjectTeamInput) (*ProjectTeam, error)
	Get(ctx context.Context, id uuid.UUID) (*ProjectTeam, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*ProjectTeam, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateProjectTeamInput) (*ProjectTeam, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
