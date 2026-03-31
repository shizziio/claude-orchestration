import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight, Loader2, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useProjectTeams } from "./hooks/use-devflow";
import type { ProjectTeam, CreateProjectTeamInput } from "@/types/devflow";

interface Props {
  projectId: string;
}

function TeamForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Partial<CreateProjectTeamInput>;
  onSubmit: (data: CreateProjectTeamInput) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("devflow");
  const [teamName, setTeamName] = useState(initial?.team_name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [configJson, setConfigJson] = useState(
    initial?.team_config ? JSON.stringify(initial.team_config, null, 2) : "{}",
  );
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    if (!teamName.trim()) return;
    let parsed: Record<string, unknown>;
    try {
      parsed = JSON.parse(configJson);
      setJsonError(null);
    } catch {
      setJsonError(t("teams.invalidJson"));
      return;
    }
    setSaving(true);
    try {
      await onSubmit({
        team_name: teamName,
        description: description || undefined,
        team_config: parsed,
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <input
        value={teamName}
        onChange={(e) => setTeamName(e.target.value)}
        placeholder={t("teams.teamName")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <input
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder={t("teams.description")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <div>
        <label className="text-xs text-muted-foreground">{t("teams.teamConfig")}</label>
        <Textarea
          value={configJson}
          onChange={(e) => {
            setConfigJson(e.target.value);
            setJsonError(null);
          }}
          className="font-mono min-h-32 resize-y mt-1"
        />
        {jsonError && <p className="text-xs text-red-500 mt-1">{jsonError}</p>}
      </div>
      <div className="flex gap-2 justify-end">
        <Button variant="outline" size="sm" onClick={onCancel} disabled={saving}>
          {t("form.cancel")}
        </Button>
        <Button size="sm" onClick={handleSubmit} disabled={saving || !teamName.trim()}>
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : null}
          {t("form.save")}
        </Button>
      </div>
    </div>
  );
}

function TeamCard({
  team,
  onEdit,
  onDelete,
}: {
  team: ProjectTeam;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-lg border">
      <div
        className="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-muted/40"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
        )}
        <span className="text-sm font-medium flex-1 truncate">{team.team_name}</span>
        {team.description && (
          <span className="text-xs text-muted-foreground truncate max-w-48 hidden sm:inline">
            {team.description}
          </span>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={(e) => { e.stopPropagation(); onEdit(); }}
        >
          <Pencil className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
          onClick={(e) => { e.stopPropagation(); onDelete(); }}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      {expanded && (
        <div className="px-3 pb-3 border-t">
          {team.description && (
            <p className="text-xs text-muted-foreground mt-2 mb-2">{team.description}</p>
          )}
          <pre className="text-xs font-mono whitespace-pre-wrap break-words mt-2 text-foreground/90 leading-relaxed bg-muted/40 rounded p-2">
            {JSON.stringify(team.team_config, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

export function ProjectTeamsEditor({ projectId }: Props) {
  const { t } = useTranslation("devflow");
  const { teams, loading, createTeam, updateTeam, deleteTeam } = useProjectTeams(projectId);
  const showSkeleton = useDeferredLoading(loading && teams.length === 0);
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  return (
    <section>
      <h2 className="text-sm font-semibold mb-2">{t("teams.title")}</h2>

      {showSkeleton ? (
        <TableSkeleton rows={2} />
      ) : (
        <div className="space-y-2">
          {teams.length === 0 && !adding && (
            <EmptyState
              icon={Users}
              title={t("teams.noTeams")}
              description={t("teams.noTeamsDesc")}
            />
          )}
          {teams.map((team) =>
            editingId === team.id ? (
              <TeamForm
                key={team.id}
                initial={{
                  team_name: team.team_name,
                  description: team.description ?? undefined,
                  team_config: team.team_config,
                }}
                onSubmit={async (data) => {
                  await updateTeam(team.id, data);
                  setEditingId(null);
                }}
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <TeamCard
                key={team.id}
                team={team}
                onEdit={() => { setEditingId(team.id); setAdding(false); }}
                onDelete={() => deleteTeam(team.id)}
              />
            ),
          )}
          {adding && (
            <TeamForm
              onSubmit={async (data) => {
                await createTeam(data);
                setAdding(false);
              }}
              onCancel={() => setAdding(false)}
            />
          )}
          {!adding && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => { setAdding(true); setEditingId(null); }}
              className="gap-1"
            >
              <Plus className="h-3.5 w-3.5" /> {t("teams.add")}
            </Button>
          )}
        </div>
      )}
    </section>
  );
}
