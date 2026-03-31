import { useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import i18next from "i18next";
import { useHttp } from "@/hooks/use-ws";
import { queryKeys } from "@/lib/query-keys";
import { toast } from "@/stores/use-toast-store";
import type {
  ProjectContext,
  TaskContext,
  TaskContextRef,
  CreateProjectContextInput,
  CreateTaskContextInput,
} from "@/types/devflow";

// ---- Project Context ----

export function useProjectContext(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: entries = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.projectContext(projectId),
    queryFn: () =>
      http.get<ProjectContext[]>(`/v1/devflow/projects/${projectId}/context`),
    enabled: !!projectId,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.projectContext(projectId) }),
    [qc, projectId],
  );

  const createEntry = useCallback(
    async (data: CreateProjectContextInput) => {
      try {
        const res = await http.post<ProjectContext>(
          `/v1/devflow/projects/${projectId}/context`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.contextCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const updateEntry = useCallback(
    async (id: string, data: Partial<CreateProjectContextInput>) => {
      try {
        const res = await http.put<ProjectContext>(
          `/v1/devflow/projects/${projectId}/context/${id}`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.contextUpdated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedUpdate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const deleteEntry = useCallback(
    async (id: string) => {
      try {
        await http.delete(`/v1/devflow/projects/${projectId}/context/${id}`);
        await invalidate();
        toast.success(i18next.t("devflow:toast.contextDeleted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedDelete"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  return { entries, loading, refresh: invalidate, createEntry, updateEntry, deleteEntry };
}

// ---- Task Context ----

export function useTaskContext(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: docs = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.taskContext(projectId),
    queryFn: () =>
      http.get<TaskContext[]>(`/v1/devflow/projects/${projectId}/task-context`),
    enabled: !!projectId,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.taskContext(projectId) }),
    [qc, projectId],
  );

  const createDoc = useCallback(
    async (data: CreateTaskContextInput) => {
      try {
        const res = await http.post<TaskContext>(
          `/v1/devflow/projects/${projectId}/task-context`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.taskContextCreated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const updateDoc = useCallback(
    async (id: string, data: Partial<CreateTaskContextInput>) => {
      try {
        const res = await http.put<TaskContext>(
          `/v1/devflow/projects/${projectId}/task-context/${id}`,
          data,
        );
        await invalidate();
        toast.success(i18next.t("devflow:toast.taskContextUpdated"));
        return res;
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedUpdate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  const deleteDoc = useCallback(
    async (id: string) => {
      try {
        await http.delete(`/v1/devflow/projects/${projectId}/task-context/${id}`);
        await invalidate();
        toast.success(i18next.t("devflow:toast.taskContextDeleted"));
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedDelete"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, invalidate],
  );

  return { docs, loading, refresh: invalidate, createDoc, updateDoc, deleteDoc };
}

// ---- Claude MD Preview ----

export function useClaudeMdPreview(projectId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data, isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.claudeMdPreview(projectId),
    queryFn: () =>
      http.get<{ content: string }>(`/v1/devflow/projects/${projectId}/claude-md/preview`),
    enabled: !!projectId,
    staleTime: 10_000,
  });

  const refresh = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.claudeMdPreview(projectId) }),
    [qc, projectId],
  );

  return { content: data?.content ?? "", loading, refresh };
}

// ---- Context Refs ----

export function useContextRefs(projectId: string, runId: string) {
  const http = useHttp();
  const qc = useQueryClient();

  const { data: refs = [], isLoading: loading } = useQuery({
    queryKey: queryKeys.devflow.contextRefs(projectId, runId),
    queryFn: () =>
      http.get<TaskContextRef[]>(`/v1/devflow/projects/${projectId}/runs/${runId}/context-refs`),
    enabled: !!projectId && !!runId,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: queryKeys.devflow.contextRefs(projectId, runId) }),
    [qc, projectId, runId],
  );

  const attachContexts = useCallback(
    async (contextIds: string[]) => {
      try {
        await http.post(
          `/v1/devflow/projects/${projectId}/runs/${runId}/context-refs`,
          { context_ids: contextIds },
        );
        await invalidate();
      } catch (err) {
        toast.error(i18next.t("devflow:toast.failedCreate"), err instanceof Error ? err.message : "");
        throw err;
      }
    },
    [http, projectId, runId, invalidate],
  );

  return { refs, loading, refresh: invalidate, attachContexts };
}
