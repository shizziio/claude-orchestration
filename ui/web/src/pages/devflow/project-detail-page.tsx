import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft, GitBranch, Play, Plus, RefreshCw, Download,
  CheckCircle, XCircle, Clock, Loader2, ExternalLink, Terminal, X, Pencil, FileText, Square,
  Code, ScrollText,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { formatRelativeTime } from "@/lib/format";
import { useProject, useEnvironments, useRuns, useRunLog, useGitCredentials } from "./hooks/use-devflow";
import { ProjectStatsCard } from "./project-stats-card";
import { ProjectEditDialog } from "./project-edit-dialog";
import { ProjectContextEditor } from "./project-context-editor";
import { ProjectTeamsEditor } from "./project-teams-editor";
import { TaskContextSelector } from "./task-context-selector";
import { TaskContextEditor } from "./task-context-editor";
import { ClaudeMdPreview } from "./claude-md-preview";
import { ComposeLogViewer } from "./compose-log-viewer";
import { useHttp } from "@/hooks/use-ws";
import { toast } from "@/stores/use-toast-store";
import i18next from "i18next";

function formatDurationMs(ms: number): string {
  const totalSeconds = Math.round(ms / 1000);
  if (totalSeconds < 60) return `${totalSeconds}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
}

interface Props {
  projectId: string;
  onBack: () => void;
}

function RunStatusIcon({ status }: { status: string }) {
  if (status === "running") return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />;
  if (status === "completed") return <CheckCircle className="h-4 w-4 text-green-500" />;
  if (status === "failed") return <XCircle className="h-4 w-4 text-red-500" />;
  return <Clock className="h-4 w-4 text-muted-foreground" />;
}

function RunStatusBadge({ status }: { status: string }) {
  const variant =
    status === "completed" ? "default" :
    status === "failed" ? "destructive" :
    status === "running" ? "secondary" : "outline";
  return <Badge variant={variant}>{status}</Badge>;
}

interface CreateRunDialogProps {
  projectId: string;
  onSubmit: (task: string, branch?: string, contextIds?: string[]) => Promise<void>;
  defaultBranch: string;
}

function CreateRunDialog({ projectId, onSubmit, defaultBranch }: CreateRunDialogProps) {
  const { t } = useTranslation("devflow");
  const [open, setOpen] = useState(false);
  const [task, setTask] = useState("");
  const [branch, setBranch] = useState(defaultBranch);
  const [selectedContextIds, setSelectedContextIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    if (!task.trim()) return;
    setLoading(true);
    try {
      await onSubmit(task, branch || undefined, selectedContextIds.length > 0 ? selectedContextIds : undefined);
      setTask("");
      setBranch(defaultBranch);
      setSelectedContextIds([]);
      setOpen(false);
    } finally {
      setLoading(false);
    }
  };

  if (!open) {
    return (
      <Button onClick={() => setOpen(true)} className="gap-1">
        <Plus className="h-4 w-4" /> {t("newRun")}
      </Button>
    );
  }

  return (
    <div className="rounded-lg border p-4 space-y-3">
      <p className="text-sm font-medium">{t("form.taskDescription")}</p>
      <textarea
        value={task}
        onChange={(e) => setTask(e.target.value)}
        placeholder={t("form.taskPlaceholder")}
        className="w-full rounded-md border bg-background px-3 py-2 text-base md:text-sm resize-none h-24 font-mono"
      />
      <div className="flex items-center gap-2">
        <label className="text-sm text-muted-foreground shrink-0">{t("form.branch")}</label>
        <input
          value={branch}
          onChange={(e) => setBranch(e.target.value)}
          className="flex-1 rounded-md border bg-background px-3 py-1.5 text-base md:text-sm font-mono"
        />
      </div>
      <TaskContextSelector
        projectId={projectId}
        selectedIds={selectedContextIds}
        onChange={setSelectedContextIds}
      />
      <div className="flex gap-2 justify-end">
        <Button variant="outline" onClick={() => setOpen(false)} disabled={loading}>{t("form.cancel")}</Button>
        <Button onClick={handleSubmit} disabled={loading || !task.trim()}>
          {loading ? t("form.saving") : t("form.startRun")}
        </Button>
      </div>
    </div>
  );
}

// ---- Run Log Drawer ----

interface RunLogDrawerProps {
  projectId: string;
  runId: string;
  status: string;
  task: string;
  onClose: () => void;
}

function RunLogDrawer({ projectId, runId, status, task, onClose }: RunLogDrawerProps) {
  const { log, complete } = useRunLog(projectId, runId, status);
  const isLive = status === "running" || status === "pending";

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background/80 backdrop-blur-sm">
      <div className="flex flex-col h-full max-w-4xl mx-auto w-full shadow-2xl bg-background border-x">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b shrink-0">
          <Terminal className="h-4 w-4 text-muted-foreground" />
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">{task}</p>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              {isLive && !complete && <Loader2 className="h-3 w-3 animate-spin" />}
              {isLive && !complete ? "running…" : complete ? "done" : status}
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} className="h-7 w-7 p-0 shrink-0">
            <X className="h-4 w-4" />
          </Button>
        </div>
        {/* Log output */}
        <div className="flex-1 overflow-y-auto p-4">
          {log ? (
            <pre className="text-xs font-mono whitespace-pre-wrap break-words text-foreground/90 leading-relaxed">
              {log}
            </pre>
          ) : (
            <p className="text-sm text-muted-foreground italic">
              {isLive ? "Waiting for output…" : "No log available."}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ---- Main page ----

export function ProjectDetailPage({ projectId, onBack }: Props) {
  const { t } = useTranslation("devflow");
  const http = useHttp();
  const { project, loading: pLoading, updateProject } = useProject(projectId);
  const { environments, loading: eLoading, startEnv, stopEnv } = useEnvironments(projectId);
  const { runs, loading: rLoading, createRun, startRun, retryRun } = useRuns(projectId);
  const { credentials } = useGitCredentials();
  const [gitLoading, setGitLoading] = useState<string | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [selectedRun, setSelectedRun] = useState<{ id: string; status: string } | null>(null);
  const [claudeMdOpen, setClaudeMdOpen] = useState(false);
  const [envActionLoading, setEnvActionLoading] = useState<string | null>(null);
  const [logViewerEnv, setLogViewerEnv] = useState<{ id: string; name: string } | null>(null);
  const [codeServerLoading, setCodeServerLoading] = useState<string | null>(null);
  const [retryLoading, setRetryLoading] = useState<string | null>(null);

  const showRunSkeleton = useDeferredLoading(rLoading && runs.length === 0);
  const showEnvSkeleton = useDeferredLoading(eLoading && environments.length === 0);

  const handleClone = async () => {
    setGitLoading("clone");
    try {
      await http.post(`/v1/devflow/projects/${projectId}/clone`, {});
      toast.success(i18next.t("devflow:toast.cloned"));
    } catch (err) {
      toast.error(i18next.t("devflow:toast.failedClone"), err instanceof Error ? err.message : "");
    } finally {
      setGitLoading(null);
    }
  };

  const handlePull = async () => {
    setGitLoading("pull");
    try {
      await http.post(`/v1/devflow/projects/${projectId}/pull`, {});
      toast.success(i18next.t("devflow:toast.pulled"));
    } catch (err) {
      toast.error(i18next.t("devflow:toast.failedPull"), err instanceof Error ? err.message : "");
    } finally {
      setGitLoading(null);
    }
  };

  const handleCreateAndStartRun = async (taskDescription: string, branch?: string, contextIds?: string[]) => {
    const run = await createRun({ task_description: taskDescription, branch });
    if (contextIds && contextIds.length > 0) {
      try {
        await http.post(`/v1/devflow/projects/${projectId}/runs/${run.id}/context-refs`, {
          context_ids: contextIds,
        });
      } catch {
        // Non-fatal: run was created, context attach failed
      }
    }
    await startRun(run.id);
  };

  const handleCodeServer = async (envId: string) => {
    setCodeServerLoading(envId);
    try {
      const res = await http.post<{ running: boolean; port: number; url: string }>(
        `/v1/devflow/projects/${projectId}/environments/${envId}/code-server/start`,
        {},
      );
      if (res.url) {
        window.open(res.url, "_blank");
        toast.success(i18next.t("devflow:toast.codeServerStarted"));
      }
    } catch (err) {
      toast.error(i18next.t("devflow:toast.failedCodeServer"), err instanceof Error ? err.message : "");
    } finally {
      setCodeServerLoading(null);
    }
  };

  if (pLoading && !project) {
    return (
      <div className="p-4 sm:p-6 pb-10">
        <TableSkeleton rows={3} />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="p-4 sm:p-6">
        <Button variant="ghost" onClick={onBack} className="gap-1 mb-4">
          <ArrowLeft className="h-4 w-4" /> {t("back")}
        </Button>
        <p className="text-muted-foreground">{t("projectNotFound")}</p>
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6 pb-10 space-y-6">
      <div>
        <Button variant="ghost" onClick={onBack} className="gap-1 mb-2 -ml-2">
          <ArrowLeft className="h-4 w-4" /> {t("back")}
        </Button>
        <PageHeader
          title={project.name}
          description={project.description ?? undefined}
          actions={
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setEditOpen(true)} className="gap-1">
                <Pencil className="h-3.5 w-3.5" />
                {t("form.editProject")}
              </Button>
              <Button variant="outline" size="sm" onClick={handleClone} disabled={!!gitLoading} className="gap-1">
                {gitLoading === "clone" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
                {t("clone")}
              </Button>
              <Button variant="outline" size="sm" onClick={handlePull} disabled={!!gitLoading} className="gap-1">
                {gitLoading === "pull" ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                {t("pull")}
              </Button>
            </div>
          }
        />
        {/* Project meta */}
        <div className="flex flex-wrap gap-3 mt-2 text-sm text-muted-foreground">
          {project.repo_url && (
            <a
              href={project.repo_url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 text-blue-500 hover:underline"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              {project.repo_url.replace(/^https?:\/\//, "").slice(0, 50)}
            </a>
          )}
          <span className="flex items-center gap-1">
            <GitBranch className="h-3.5 w-3.5" />
            {project.default_branch}
          </span>
          {project.workspace_path && (
            <span className="font-mono text-xs bg-muted px-2 py-0.5 rounded">
              {project.workspace_path}
            </span>
          )}
          <Badge variant={project.status === "active" ? "default" : "secondary"}>{project.status}</Badge>
        </div>
      </div>

      {/* Stats */}
      <ProjectStatsCard projectId={projectId} />

      {/* Context */}
      <ProjectContextEditor projectId={projectId} />

      {/* Teams */}
      <ProjectTeamsEditor projectId={projectId} />

      {/* Task Context */}
      <TaskContextEditor projectId={projectId} />

      <div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setClaudeMdOpen(true)}
          className="gap-1"
        >
          <FileText className="h-3.5 w-3.5" />
          {t("context.previewClaudeMd")}
        </Button>
      </div>

      {/* Environments */}
      <section>
        <h2 className="text-sm font-semibold mb-2">{t("environments")}</h2>
        {showEnvSkeleton ? (
          <TableSkeleton rows={2} />
        ) : environments.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("noEnvironments")}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-[600px] w-full text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="pb-2 pr-4 font-medium">{t("col.name")}</th>
                  <th className="pb-2 pr-4 font-medium">{t("col.branch")}</th>
                  <th className="pb-2 pr-4 font-medium">{t("col.status")}</th>
                  <th className="pb-2 pr-4 font-medium">Compose</th>
                  <th className="pb-2 pr-4 font-medium">{t("col.updated")}</th>
                  <th className="pb-2 pr-4 font-medium" />
                </tr>
              </thead>
              <tbody>
                {environments.map((env) => {
                  const cs = env.compose_status ?? "stopped";
                  const isBusy = envActionLoading === env.id;
                  return (
                    <tr key={env.id} className="border-b hover:bg-muted/40">
                      <td className="py-2 pr-4">
                        <span className="font-medium">{env.name}</span>
                        <span className="ml-2 text-xs text-muted-foreground">{env.slug}</span>
                      </td>
                      <td className="py-2 pr-4 text-xs font-mono">{env.branch ?? project.default_branch}</td>
                      <td className="py-2 pr-4">
                        <Badge variant="outline">{env.status}</Badge>
                      </td>
                      <td className="py-2 pr-4">
                        <Badge
                          variant={cs === "running" ? "default" : cs === "error" ? "destructive" : "outline"}
                          className={
                            cs === "running" ? "bg-green-600 hover:bg-green-600" :
                            cs === "starting" ? "border-yellow-500 text-yellow-600" :
                            undefined
                          }
                        >
                          {t(`env.composeStatus.${cs}`, cs)}
                        </Badge>
                      </td>
                      <td className="py-2 pr-4 text-xs text-muted-foreground">
                        {formatRelativeTime(env.updated_at)}
                      </td>
                      <td className="py-2 pr-4">
                        <div className="flex items-center gap-1">
                          {cs === "starting" ? (
                            <Loader2 className="h-4 w-4 animate-spin text-yellow-500" />
                          ) : cs === "running" ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              disabled={isBusy}
                              className="h-7 w-7 p-0"
                              onClick={async () => {
                                setEnvActionLoading(env.id);
                                try { await stopEnv(env.id); } finally { setEnvActionLoading(null); }
                              }}
                              title={t("env.stop")}
                            >
                              {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
                            </Button>
                          ) : (
                            <Button
                              variant="ghost"
                              size="sm"
                              disabled={isBusy}
                              className="h-7 w-7 p-0"
                              onClick={async () => {
                                setEnvActionLoading(env.id);
                                try { await startEnv(env.id); } finally { setEnvActionLoading(null); }
                              }}
                              title={t("env.start")}
                            >
                              {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            </Button>
                          )}
                          {cs === "running" && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 w-7 p-0"
                              onClick={() => setLogViewerEnv({ id: env.id, name: env.name })}
                              title={t("env.logs")}
                            >
                              <ScrollText className="h-4 w-4" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={codeServerLoading === env.id}
                            className="h-7 p-0 gap-1"
                            onClick={() => {
                              if (env.code_server_port) {
                                window.open(`http://localhost:${env.code_server_port}`, "_blank");
                              } else {
                                handleCodeServer(env.id);
                              }
                            }}
                            title={env.code_server_port ? t("env.openIDE") : t("env.codeServer")}
                          >
                            {codeServerLoading === env.id ? (
                              <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                              <Code className="h-4 w-4" />
                            )}
                            {env.code_server_port && (
                              <span className="text-[10px] text-muted-foreground">{env.code_server_port}</span>
                            )}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Runs */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold">{t("runs")}</h2>
        </div>

        <CreateRunDialog
          projectId={projectId}
          onSubmit={handleCreateAndStartRun}
          defaultBranch={project.default_branch}
        />

        <div className="mt-4">
          {showRunSkeleton ? (
            <TableSkeleton rows={3} />
          ) : runs.length === 0 ? (
            <EmptyState
              icon={Play}
              title={t("noRuns")}
              description={t("noRunsDesc")}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-[600px] w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-muted-foreground">
                    <th className="pb-2 pr-4 font-medium">{t("col.status")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.task")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.branch")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.cost")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.duration")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.started")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.completed")}</th>
                    <th className="pb-2 pr-4 font-medium" />
                  </tr>
                </thead>
                <tbody>
                  {runs.map((run) => (
                    <tr
                      key={run.id}
                      className="border-b hover:bg-muted/40 cursor-pointer"
                      onClick={() => setSelectedRun({ id: run.id, status: run.status })}
                    >
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-2">
                          <RunStatusIcon status={run.status} />
                          <RunStatusBadge status={run.status} />
                        </div>
                      </td>
                      <td className="py-3 pr-4 max-w-xs">
                        <p className="truncate">{run.task_description}</p>
                        {run.error_message && (
                          <p className="text-xs text-red-500 mt-0.5 truncate">{run.error_message}</p>
                        )}
                        {run.result_summary && (
                          <p className="text-xs text-muted-foreground mt-0.5 truncate">{run.result_summary}</p>
                        )}
                      </td>
                      <td className="py-3 pr-4 text-xs font-mono">
                        {run.branch ?? project.default_branch}
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {run.cost_usd != null ? `$${run.cost_usd.toFixed(2)}` : "\u2014"}
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {run.duration_ms != null ? formatDurationMs(run.duration_ms) : "\u2014"}
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {run.started_at ? formatRelativeTime(run.started_at) : "\u2014"}
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {run.completed_at ? formatRelativeTime(run.completed_at) : "\u2014"}
                      </td>
                      <td className="py-3 pr-4">
                        {(run.status === "failed" || run.status === "completed") && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 w-7 p-0"
                            disabled={retryLoading === run.id}
                            title={t("retry")}
                            onClick={async (e) => {
                              e.stopPropagation();
                              setRetryLoading(run.id);
                              try { await retryRun(run.id); } finally { setRetryLoading(null); }
                            }}
                          >
                            {retryLoading === run.id
                              ? <Loader2 className="h-4 w-4 animate-spin" />
                              : <RefreshCw className="h-4 w-4" />}
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>

      {/* Edit dialog */}
      <ProjectEditDialog
        open={editOpen}
        project={project}
        credentials={credentials}
        onClose={() => setEditOpen(false)}
        onSubmit={async (data) => {
          await updateProject(data);
        }}
      />

      {/* Claude MD preview drawer */}
      {claudeMdOpen && (
        <ClaudeMdPreview projectId={projectId} onClose={() => setClaudeMdOpen(false)} />
      )}

      {/* Compose log viewer drawer */}
      {logViewerEnv && (
        <ComposeLogViewer
          projectId={projectId}
          envId={logViewerEnv.id}
          envName={logViewerEnv.name}
          onClose={() => setLogViewerEnv(null)}
        />
      )}

      {/* Run log drawer */}
      {selectedRun && (() => {
        const liveRun = runs.find((r) => r.id === selectedRun.id);
        const currentStatus = liveRun?.status ?? selectedRun.status;
        const task = liveRun?.task_description ?? "";
        return (
          <RunLogDrawer
            projectId={projectId}
            runId={selectedRun.id}
            status={currentStatus}
            task={task}
            onClose={() => setSelectedRun(null)}
          />
        );
      })()}
    </div>
  );
}
