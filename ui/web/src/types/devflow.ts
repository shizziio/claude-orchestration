export interface GitCredential {
  id: string;
  tenant_id: string;
  user_id: string;
  label: string;
  provider: string; // github | gitlab | bitbucket | custom
  host: string | null;
  auth_type: string; // ssh_key | token
  public_key: string | null;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  description: string | null;
  repo_url: string | null;
  default_branch: string;
  git_credential_id: string | null;
  workspace_path: string | null;
  docker_compose_file: string | null;
  status: string; // active | archived
  settings: Record<string, unknown>;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Environment {
  id: string;
  tenant_id: string;
  project_id: string;
  name: string;
  slug: string;
  status: string; // dormant | coding | running
  branch: string | null;
  docker_compose_override: string | null;
  compose_status: string; // "stopped" | "starting" | "running" | "error"
  code_server_port: number | null;
  preview_port: number | null;
  port_bindings: Record<string, number>;
  created_at: string;
  updated_at: string;
}

export interface DevflowRun {
  id: string;
  tenant_id: string;
  project_id: string;
  environment_id: string | null;
  task_description: string;
  context_prompt: string | null;
  branch: string | null;
  status: string; // pending | running | completed | failed
  claude_session_id: string | null;
  result_summary: string | null;
  error_message: string | null;
  created_by: string;
  started_at: string | null;
  completed_at: string | null;
  cost_usd: number | null;
  duration_ms: number | null;
  created_at: string;
  updated_at: string;
}

export interface ProjectStats {
  total_runs: number;
  completed_runs: number;
  failed_runs: number;
  running_runs: number;
  total_cost_usd: number;
  avg_duration_ms: number;
  success_rate: number;
}

export interface CreateProjectInput {
  name: string;
  slug: string;
  description?: string;
  repo_url?: string;
  default_branch?: string;
  git_credential_id?: string;
  docker_compose_file?: string;
}

export interface CreateGitCredentialInput {
  label: string;
  provider: string;
  host?: string;
  auth_type: string;
  public_key?: string;
  private_key?: string;
  token?: string;
}

export interface CreateRunInput {
  task_description: string;
  context_prompt?: string;
  branch?: string;
  environment_id?: string;
}

export interface ProjectContext {
  id: string;
  tenant_id: string;
  project_id: string;
  doc_type: string; // "rules" | "index" | "structure"
  title: string;
  content: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface TaskContext {
  id: string;
  tenant_id: string;
  project_id: string;
  title: string;
  content: string;
  tags: string[];
  file_path: string | null;
  created_at: string;
  updated_at: string;
}

export interface TaskContextRef {
  id: string;
  run_id: string;
  task_context_id: string;
  created_at: string;
}

export interface CreateProjectContextInput {
  doc_type: string;
  title: string;
  content: string;
  sort_order?: number;
}

export interface CreateTaskContextInput {
  title: string;
  content: string;
  tags?: string[];
  file_path?: string;
}

export interface ProjectTeam {
  id: string;
  tenant_id: string;
  project_id: string;
  team_name: string;
  description: string | null;
  team_config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectTeamInput {
  team_name: string;
  description?: string;
  team_config?: Record<string, unknown>;
}
