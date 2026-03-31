export const queryKeys = {
  apiKeys: {
    all: ["apiKeys"] as const,
  },
  providers: {
    all: ["providers"] as const,
    models: (providerId: string) => ["providers", providerId, "models"] as const,
  },
  agents: {
    all: ["agents"] as const,
    detail: (id: string) => ["agents", id] as const,
    files: (agentKey: string) => ["agents", agentKey, "files"] as const,
    links: (agentId: string) => ["agents", agentId, "links"] as const,
    instances: (agentId: string) => ["agents", agentId, "instances"] as const,
  },
  sessions: {
    all: ["sessions"] as const,
    list: (params: Record<string, unknown>) => ["sessions", params] as const,
  },
  traces: {
    all: ["traces"] as const,
    list: (params: Record<string, unknown>) => ["traces", params] as const,
  },
  cliCredentials: {
    all: ["cliCredentials"] as const,
  },
  mcp: {
    all: ["mcp"] as const,
  },
  channels: {
    all: ["channels"] as const,
    list: (params: Record<string, unknown>) => ["channels", params] as const,
    detail: (id: string) => ["channels", "detail", id] as const,
  },
  contacts: {
    all: ["contacts"] as const,
    list: (params: Record<string, unknown>) => ["contacts", params] as const,
    resolve: (ids: string) => ["contacts", "resolve", ids] as const,
  },
  skills: {
    all: ["skills"] as const,
    agentGrants: (agentId: string) => ["skills", "agent", agentId] as const,
    runtimes: ["skills", "runtimes"] as const,
  },
  cron: {
    all: ["cron"] as const,
  },
  builtinTools: {
    all: ["builtinTools"] as const,
  },
  config: {
    all: ["config"] as const,
  },
  tts: {
    all: ["tts"] as const,
  },
  usage: {
    all: ["usage"] as const,
    records: (params: Record<string, unknown>) => ["usage", "records", params] as const,
  },
  teams: {
    all: ["teams"] as const,
    detail: (id: string) => ["teams", id] as const,
  },
  memory: {
    all: ["memory"] as const,
    list: (params: Record<string, unknown>) => ["memory", params] as const,
  },
  packages: {
    all: ["packages"] as const,
    runtimes: ["packages", "runtimes"] as const,
  },
  tenantUsers: {
    all: ["tenantUsers"] as const,
  },
  tenants: {
    all: ["tenants"] as const,
    detail: (tenantId: string) => ["tenants", tenantId] as const,
    users: (tenantId: string) => ["tenants", tenantId, "users"] as const,
  },
  kg: {
    all: ["kg"] as const,
    list: (params: Record<string, unknown>) => ["kg", params] as const,
    stats: (agentId: string, userId?: string) => ["kg", "stats", agentId, userId] as const,
    graph: (agentId: string, userId?: string) => ["kg", "graph", agentId, userId] as const,
  },
  devflow: {
    projects: ["devflow", "projects"] as const,
    project: (id: string) => ["devflow", "projects", id] as const,
    gitCredentials: ["devflow", "git-credentials"] as const,
    environments: (projectId: string) => ["devflow", "projects", projectId, "environments"] as const,
    runs: (projectId: string) => ["devflow", "projects", projectId, "runs"] as const,
    run: (projectId: string, runId: string) => ["devflow", "projects", projectId, "runs", runId] as const,
    projectContext: (projectId: string) => ["devflow", "project-context", projectId] as const,
    taskContext: (projectId: string) => ["devflow", "task-context", projectId] as const,
    contextRefs: (projectId: string, runId: string) => ["devflow", "context-refs", projectId, runId] as const,
    teams: (projectId: string) => ["devflow", "teams", projectId] as const,
    claudeMdPreview: (projectId: string) => ["devflow", "claude-md-preview", projectId] as const,
    stats: (projectId: string) => ["devflow", "stats", projectId] as const,
    webhooks: (projectId: string) => ["devflow", "webhooks", projectId] as const,
  },
};
