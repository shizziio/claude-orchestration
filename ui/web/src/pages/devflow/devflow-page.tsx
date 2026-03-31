import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";
import { FolderGit2, Plus, Key, Trash2, GitBranch, ExternalLink, Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { SearchInput } from "@/components/shared/search-input";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDeleteDialog } from "@/components/shared/confirm-delete-dialog";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { formatRelativeTime } from "@/lib/format";
import { ROUTES } from "@/lib/constants";
import { useProjects, useGitCredentials } from "./hooks/use-devflow";
import { ProjectFormDialog } from "./project-form-dialog";
import { GitCredentialFormDialog } from "./git-credential-form-dialog";
import { GitCredentialEditDialog } from "./git-credential-edit-dialog";
import { ProjectDetailPage } from "./project-detail-page";
import type { Project, GitCredential } from "@/types/devflow";

export function DevflowPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  if (id) {
    return <ProjectDetailPage projectId={id} onBack={() => navigate(ROUTES.DEVFLOW)} />;
  }
  return <DevflowListView />;
}

function DevflowListView() {
  const { t } = useTranslation("devflow");
  const navigate = useNavigate();
  const [tab, setTab] = useState<"projects" | "credentials">("projects");
  const [search, setSearch] = useState("");
  const [projectFormOpen, setProjectFormOpen] = useState(false);
  const [credFormOpen, setCredFormOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null);
  const [deleteCred, setDeleteCred] = useState<GitCredential | null>(null);
  const [editCred, setEditCred] = useState<GitCredential | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const { projects, loading: pLoading, createProject, deleteProject } = useProjects();
  const { credentials, loading: cLoading, createCredential, updateCredential, deleteCredential } = useGitCredentials();

  const showProjSkeleton = useDeferredLoading(pLoading && projects.length === 0);
  const showCredSkeleton = useDeferredLoading(cLoading && credentials.length === 0);

  const filteredProjects = projects.filter(
    (p) =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.slug.toLowerCase().includes(search.toLowerCase()),
  );

  const filteredCreds = credentials.filter((c) =>
    c.label.toLowerCase().includes(search.toLowerCase()),
  );

  const handleDeleteProject = () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    deleteProject(deleteTarget.id)
      .then(() => setDeleteTarget(null))
      .finally(() => setDeleteLoading(false));
  };

  const handleDeleteCred = () => {
    if (!deleteCred) return;
    setDeleteLoading(true);
    deleteCredential(deleteCred.id)
      .then(() => setDeleteCred(null))
      .finally(() => setDeleteLoading(false));
  };

  return (
    <div className="p-4 sm:p-6 pb-10">
      <PageHeader
        title={t("title")}
        description={t("description")}
        actions={
          tab === "projects" ? (
            <Button onClick={() => setProjectFormOpen(true)} className="gap-1">
              <Plus className="h-4 w-4" /> {t("addProject")}
            </Button>
          ) : (
            <Button onClick={() => setCredFormOpen(true)} className="gap-1">
              <Plus className="h-4 w-4" /> {t("addCredential")}
            </Button>
          )
        }
      />

      {/* Tabs */}
      <div className="mb-4 flex gap-1 border-b">
        {(["projects", "credentials"] as const).map((t2) => (
          <button
            key={t2}
            onClick={() => { setTab(t2); setSearch(""); }}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              tab === t2
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {t2 === "projects" ? t("tabs.projects") : t("tabs.credentials")}
          </button>
        ))}
      </div>

      <SearchInput
        value={search}
        onChange={setSearch}
        placeholder={t("search")}
        className="mb-4 max-w-sm"
      />

      {/* Projects tab */}
      {tab === "projects" && (
        <>
          {showProjSkeleton ? (
            <TableSkeleton rows={4} />
          ) : filteredProjects.length === 0 ? (
            <EmptyState
              icon={FolderGit2}
              title={t("noProjects")}
              description={t("noProjectsDesc")}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-[600px] w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-muted-foreground">
                    <th className="pb-2 pr-4 font-medium">{t("col.name")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.repo")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.branch")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.status")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.updated")}</th>
                    <th className="pb-2 font-medium" />
                  </tr>
                </thead>
                <tbody>
                  {filteredProjects.map((p) => (
                    <tr
                      key={p.id}
                      className="border-b hover:bg-muted/40 cursor-pointer"
                      onClick={() => navigate(`/devflow/${p.id}`)}
                    >
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-2">
                          <FolderGit2 className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{p.name}</span>
                          <span className="text-xs text-muted-foreground">{p.slug}</span>
                        </div>
                        {p.description && (
                          <div className="text-xs text-muted-foreground mt-0.5">{p.description}</div>
                        )}
                      </td>
                      <td className="py-3 pr-4">
                        {p.repo_url ? (
                          <a
                            href={p.repo_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={(e) => e.stopPropagation()}
                            className="flex items-center gap-1 text-xs text-blue-500 hover:underline"
                          >
                            <ExternalLink className="h-3 w-3" />
                            {p.repo_url.replace(/^https?:\/\//, "").slice(0, 40)}
                          </a>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </td>
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-1 text-xs">
                          <GitBranch className="h-3 w-3 text-muted-foreground" />
                          {p.default_branch}
                        </div>
                      </td>
                      <td className="py-3 pr-4">
                        <Badge variant={p.status === "active" ? "default" : "secondary"}>
                          {p.status}
                        </Badge>
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {formatRelativeTime(p.updated_at)}
                      </td>
                      <td className="py-3">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => { e.stopPropagation(); setDeleteTarget(p); }}
                          className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Credentials tab */}
      {tab === "credentials" && (
        <>
          {showCredSkeleton ? (
            <TableSkeleton rows={3} />
          ) : filteredCreds.length === 0 ? (
            <EmptyState
              icon={Key}
              title={t("noCredentials")}
              description={t("noCredentialsDesc")}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-[600px] w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-muted-foreground">
                    <th className="pb-2 pr-4 font-medium">{t("col.label")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.provider")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.authType")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.host")}</th>
                    <th className="pb-2 pr-4 font-medium">{t("col.created")}</th>
                    <th className="pb-2 font-medium" />
                  </tr>
                </thead>
                <tbody>
                  {filteredCreds.map((c) => (
                    <tr key={c.id} className="border-b hover:bg-muted/40">
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-2">
                          <Key className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{c.label}</span>
                        </div>
                      </td>
                      <td className="py-3 pr-4">
                        <Badge variant="outline">{c.provider}</Badge>
                      </td>
                      <td className="py-3 pr-4 text-xs">{c.auth_type}</td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {c.host ?? "—"}
                      </td>
                      <td className="py-3 pr-4 text-xs text-muted-foreground">
                        {formatRelativeTime(c.created_at)}
                      </td>
                      <td className="py-3">
                        <div className="flex gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditCred(c)}
                            className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setDeleteCred(c)}
                            className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      <ProjectFormDialog
        open={projectFormOpen}
        onClose={() => setProjectFormOpen(false)}
        onSubmit={async (data) => { await createProject(data); setProjectFormOpen(false); }}
        credentials={credentials}
      />

      <GitCredentialFormDialog
        open={credFormOpen}
        onClose={() => setCredFormOpen(false)}
        onSubmit={async (data) => { await createCredential(data); setCredFormOpen(false); }}
      />

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title={t("deleteProject")}
        description={t("deleteProjectDesc", { name: deleteTarget?.name })}
        confirmValue={deleteTarget?.name ?? ""}
        loading={deleteLoading}
        onConfirm={handleDeleteProject}
      />

      {editCred && (
        <GitCredentialEditDialog
          open={!!editCred}
          credential={editCred}
          onClose={() => setEditCred(null)}
          onSubmit={async (data) => { await updateCredential(editCred.id, data); setEditCred(null); }}
        />
      )}

      <ConfirmDeleteDialog
        open={!!deleteCred}
        onOpenChange={(o) => !o && setDeleteCred(null)}
        title={t("deleteCredential")}
        description={t("deleteCredentialDesc", { label: deleteCred?.label })}
        confirmValue=""
        loading={deleteLoading}
        onConfirm={handleDeleteCred}
      />
    </div>
  );
}
