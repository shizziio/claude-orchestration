import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useProjectContext } from "./hooks/use-context";
import type { ProjectContext, CreateProjectContextInput } from "@/types/devflow";

const DOC_TYPES = ["rules", "index", "structure"] as const;

interface Props {
  projectId: string;
}

function EntryForm({
  initial,
  docType,
  onSubmit,
  onCancel,
}: {
  initial?: Partial<CreateProjectContextInput>;
  docType: string;
  onSubmit: (data: CreateProjectContextInput) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("devflow");
  const [title, setTitle] = useState(initial?.title ?? "");
  const [content, setContent] = useState(initial?.content ?? "");
  const [sortOrder, setSortOrder] = useState(initial?.sort_order ?? 0);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    if (!title.trim() || !content.trim()) return;
    setSaving(true);
    try {
      await onSubmit({ doc_type: docType, title, content, sort_order: sortOrder });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={t("context.titlePlaceholder")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <Textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder={t("context.contentPlaceholder")}
        className="font-mono min-h-32 resize-y"
      />
      <div className="flex items-center gap-2">
        <label className="text-xs text-muted-foreground shrink-0">{t("context.sortOrder")}</label>
        <input
          type="number"
          value={sortOrder}
          onChange={(e) => setSortOrder(Number(e.target.value))}
          className="w-20 rounded-md border bg-background px-2 py-1 text-base md:text-sm"
        />
        <div className="flex-1" />
        <Button variant="outline" size="sm" onClick={onCancel} disabled={saving}>
          {t("form.cancel")}
        </Button>
        <Button size="sm" onClick={handleSubmit} disabled={saving || !title.trim() || !content.trim()}>
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : null}
          {t("form.save")}
        </Button>
      </div>
    </div>
  );
}

function EntryCard({
  entry,
  onEdit,
  onDelete,
}: {
  entry: ProjectContext;
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
        <span className="text-sm font-medium flex-1 truncate">{entry.title}</span>
        <span className="text-xs text-muted-foreground tabular-nums">#{entry.sort_order}</span>
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
          <pre className="text-xs font-mono whitespace-pre-wrap break-words mt-2 text-foreground/90 leading-relaxed">
            {entry.content}
          </pre>
        </div>
      )}
    </div>
  );
}

export function ProjectContextEditor({ projectId }: Props) {
  const { t } = useTranslation("devflow");
  const { entries, loading, createEntry, updateEntry, deleteEntry } = useProjectContext(projectId);
  const showSkeleton = useDeferredLoading(loading && entries.length === 0);
  const [activeTab, setActiveTab] = useState<string>("rules");
  const [addingType, setAddingType] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);

  const filtered = entries.filter((e) => e.doc_type === activeTab);

  return (
    <section>
      <h2 className="text-sm font-semibold mb-2">{t("context.title")}</h2>

      {/* Doc type tabs */}
      <div className="flex gap-1 mb-3">
        {DOC_TYPES.map((dt) => {
          const count = entries.filter((e) => e.doc_type === dt).length;
          return (
            <Button
              key={dt}
              variant={activeTab === dt ? "default" : "outline"}
              size="sm"
              onClick={() => { setActiveTab(dt); setAddingType(null); setEditingId(null); }}
              className="gap-1"
            >
              {t(`context.docType.${dt}`)}
              {count > 0 && <Badge variant="secondary" className="ml-1 text-xs px-1.5">{count}</Badge>}
            </Button>
          );
        })}
      </div>

      {showSkeleton ? (
        <TableSkeleton rows={2} />
      ) : (
        <div className="space-y-2">
          {filtered.length === 0 && !addingType && (
            <EmptyState
              icon={Plus}
              title={t("context.noEntries")}
              description={t("context.noEntriesDesc")}
            />
          )}
          {filtered
            .sort((a, b) => a.sort_order - b.sort_order)
            .map((entry) =>
              editingId === entry.id ? (
                <EntryForm
                  key={entry.id}
                  initial={{ title: entry.title, content: entry.content, sort_order: entry.sort_order }}
                  docType={activeTab}
                  onSubmit={async (data) => {
                    await updateEntry(entry.id, data);
                    setEditingId(null);
                  }}
                  onCancel={() => setEditingId(null)}
                />
              ) : (
                <EntryCard
                  key={entry.id}
                  entry={entry}
                  onEdit={() => { setEditingId(entry.id); setAddingType(null); }}
                  onDelete={() => deleteEntry(entry.id)}
                />
              ),
            )}
          {addingType === activeTab && (
            <EntryForm
              docType={activeTab}
              onSubmit={async (data) => {
                await createEntry(data);
                setAddingType(null);
              }}
              onCancel={() => setAddingType(null)}
            />
          )}
          {addingType !== activeTab && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => { setAddingType(activeTab); setEditingId(null); }}
              className="gap-1"
            >
              <Plus className="h-3.5 w-3.5" /> {t("context.add")}
            </Button>
          )}
        </div>
      )}
    </section>
  );
}
