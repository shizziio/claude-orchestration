import { useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import i18next from "i18next";
import { useHttp } from "@/hooks/use-ws";
import { queryKeys } from "@/lib/query-keys";
import { toast } from "@/stores/use-toast-store";
import type {
  Project, GitCredential, Environment, DevflowRun, ProjectTeam, ProjectStats,
  CreateProjectInput, CreateGitCredentialInput, CreateRunInput, CreateProjectTeamInput,
} from "@/types/devflow";

// ---- Projects ----

export function useProjects() {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: projects = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.projects,
    queryFn: () => http.get<Project[]>("/v1/devflow/projects"),
    staleTime: 30_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.projects }),
    [qc],
  );

  const createProject = useCallback(
    async (data: CreateProjectInput) => {
      try {
        const res = await http.post<Project>("/v1/devflow/projects", data);
        await invalidate();
        toast.success(i18next.t("devflow:toast.projectCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, invalidate],
  );

  const deleteProject = useCallback(
    async (id: string) => {
      try {
        await http.delete(`/v1/devflow/projects/${id}`);
        await invalidate();
        toast.success(i18next.t("devflow:toast.projectDeleted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedDelete"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, invalidate],
  );

  return { projects, loading, refresh: invalidate, createProject, deleteProject };
}

export function useProject(id: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: project, isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.project(id),
    queryFn: () => http.get<Project>(`/v1/devflow/projects/${id}`),
    enabled: !!id,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.project(id) }),
    [qc, id],
  );

  const updateProject = useCallback(
    async (data: Partial<CreateProjectInput & { workspace_path: string; status: string }>) => {
      try {
        const res = await http.put<Project>(`/v1/devflow/projects/${id}`, data);
        await invalidate();
        toast.success(i18next.t("devflow:toast.projectUpdated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedUpdate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, id, invalidate],
  );

  const cloneRepo = useCallback(
    async (branch?: string) => {
      try {
        const res = await http.post<{ status: string; workspace_path: string }>(
          `/v1/devflow/projects/${id}/clone`,
          { branch },
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.cloned"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedClone"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, id, invalidate],
  );

  const pullRepo = useCallback(
    async (branch?: string) => {
      try {
        await http.post(`/v1/devflow/projects/${id}/pull`, { branch });
        toast.success(i18next.t("devflow:toast.pulled"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedPull"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, id],
  );

  return { project, loading, refresh: invalidate, updateProject, cloneRepo, pullRepo };
}

// ---- Git Credentials ----

export function useGitCredentials() {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: credentials = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.gitCredentials,
    queryFn: () => http.get<GitCredential[]>("/v1/devflow/git-credentials"),
    staleTime: 60_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.gitCredentials }),
    [qc],
  );

  const createCredential = useCallback(
    async (data: CreateGitCredentialInput) => {
      try {
        const res = await http.post<GitCredential>("/v1/devflow/git-credentials", data);
        await invalidate();
        toast.success(i18next.t("devflow:toast.credentialCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, invalidate],
  );

  const updateCredential = useCallback(
    async (id: string, data: Partial<{ label: string; host: string; private_key: string; token: string }>) => {
      try {
        const res = await http.put<import("@/types/devflow").GitCredential>(`/v1/devflow/git-credentials/${id}`, data);
        await invalidate();
        toast.success(i18next.t("devflow:toast.credentialUpdated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedUpdate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, invalidate],
  );

  const deleteCredential = useCallback(
    async (id: string) => {
      try {
        await http.delete(`/v1/devflow/git-credentials/${id}`);
        await invalidate();
        toast.success(i18next.t("devflow:toast.credentialDeleted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedDelete"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, invalidate],
  );

  return { credentials, loading, refresh: invalidate, createCredential, updateCredential, deleteCredential };
}

// ---- Environments ----

export function useEnvironments(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: environments = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.environments(projectId),
    queryFn: () =>
      http.get<Environment[]>(`/v1/devflow/projects/${projectId}/environments`),
    enabled: !!projectId,
    staleTime: 15_000,
    refetchInterval: (query) => {
      const data = query.state.data as Environment[] | undefined;
      if (data?.some((e) => e.compose_status === "starting")) {
        return 5_000;
      }
      return false;
    },
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.environments(projectId) }),
    [qc, projectId],
  );

  const createEnvironment = useCallback(
    async (data: { name: string; slug: string; branch?: string }) => {
      try {
        const res = await http.post<Environment>(
          `/v1/devflow/projects/${projectId}/environments`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.envCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const startEnv = useCallback(
    async (envId: string) => {
      try {
        await http.post(`/v1/devflow/projects/${projectId}/environments/${envId}/start`, {});
        await invalidate();
        toast.success(i18next.t("devflow:toast.envStarted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedEnvStart"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const stopEnv = useCallback(
    async (envId: string) => {
      try {
        await http.post(`/v1/devflow/projects/${projectId}/environments/${envId}/stop`, {});
        await invalidate();
        toast.success(i18next.t("devflow:toast.envStopped"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedEnvStop"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  return { environments, loading, refresh: invalidate, createEnvironment, startEnv, stopEnv };
}

// ---- Runs ----

export function useRuns(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: runs = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.runs(projectId),
    queryFn: () =>
      http.get<DevflowRun[]>(`/v1/devflow/projects/${projectId}/runs`),
    enabled: !!projectId,
    staleTime: 10_000,
    refetchInterval: (query) => {
      // Auto-refresh while there are active runs
      const data = query.state.data as DevflowRun[] | undefined;
      if (data?.some((r) => r.status === "running" || r.status === "pending")) {
        return 5_000;
      }
      return false;
    },
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.runs(projectId) }),
    [qc, projectId],
  );

  const createRun = useCallback(
    async (data: CreateRunInput) => {
      try {
        const res = await http.post<DevflowRun>(
          `/v1/devflow/projects/${projectId}/runs`,
          data,
        );
        await invalidate();
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const startRun = useCallback(
    async (runId: string) => {
      try {
        await http.post(`/v1/devflow/projects/${projectId}/runs/${runId}/start`, {});
        await invalidate();
        toast.success(i18next.t("devflow:toast.runStarted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedStart"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const retryRun = useCallback(
    async (runId: string) => {
      try {
        const res = await http.post<DevflowRun>(
          `/v1/devflow/projects/${projectId}/runs/${runId}/retry`,
          {},
        );
        await invalidate();
        await qc.invalidateQueries({ queryKey: queryKeys.devflow.stats(projectId) });
        toast.success(i18next.t("devflow:toast.runRetried"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedStart"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate, qc],
  );

  return { runs, loading, refresh: invalidate, createRun, startRun, retryRun };
}

// ---- Teams ----

export function useProjectTeams(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: teams = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.teams(projectId),
    queryFn: () =>
      http.get<ProjectTeam[]>(`/v1/devflow/projects/${projectId}/teams`),
    enabled: !!projectId,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.teams(projectId) }),
    [qc, projectId],
  );

  const createTeam = useCallback(
    async (data: CreateProjectTeamInput) => {
      try {
        const res = await http.post<ProjectTeam>(
          `/v1/devflow/projects/${projectId}/teams`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.teamCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const updateTeam = useCallback(
    async (id: string, data: Partial<CreateProjectTeamInput>) => {
      try {
        const res = await http.put<ProjectTeam>(
          `/v1/devflow/projects/${projectId}/teams/${id}`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.teamUpdated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedUpdate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const deleteTeam = useCallback(
    async (id: string) => {
      try {
        await http.delete(`/v1/devflow/projects/${projectId}/teams/${id}`);
        await invalidate();
        toast.success(i18next.t("devflow:toast.teamDeleted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedDelete"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  return { teams, loading, refresh: invalidate, createTeam, updateTeam, deleteTeam };
}

// ---- Project Stats ----

export function useProjectStats(projectId: string) {
  const http = useHttp();

  const { data: stats, isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.stats(projectId),
    queryFn: () => http.get<ProjectStats>(`/v1/devflow/projects/${projectId}/stats`),
    enabled: !!projectId,
    staleTime: 15_000,
  });

  return { stats, loading };
}

export function useRunLog(projectId: string, runId: string, status: string) {
  const http = useHttp();
  const isActive = status === "running" || status === "pending";

  const { data } = useQuery({
    queryKey: queryKeys.devflow.run(projectId, runId),
    queryFn: () =>
      http.get<{ log: string; complete: boolean }>(`/v1/devflow/projects/${projectId}/runs/${runId}/log`),
    enabled: !!projectId && !!runId,
    staleTime: 0,
    refetchInterval: isActive ? 2_000 : false,
  });

  return { log: data?.log ?? "", complete: data?.complete ?? false };
}
