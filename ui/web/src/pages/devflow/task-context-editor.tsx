import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight, Loader2, FolderOpen, Tag } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useTaskContext } from "./hooks/use-context";
import type { TaskContext, CreateTaskContextInput } from "@/types/devflow";

interface Props {
  projectId: string;
}

function DocForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Partial<CreateTaskContextInput & { tags_str?: string }>;
  onSubmit: (data: CreateTaskContextInput) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("devflow");
  const [title, setTitle] = useState(initial?.title ?? "");
  const [content, setContent] = useState(initial?.content ?? "");
  const [tagsStr, setTagsStr] = useState(
    initial?.tags_str ?? (initial?.tags?.join(", ") ?? ""),
  );
  const [filePath, setFilePath] = useState(initial?.file_path ?? "");
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    if (!title.trim() || !content.trim()) return;
    setSaving(true);
    try {
      const tags = tagsStr
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean);
      await onSubmit({
        title,
        content,
        tags: tags.length > 0 ? tags : undefined,
        file_path: filePath.trim() || undefined,
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={t("context.taskContext.titleLabel")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <Textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder={t("context.taskContext.contentLabel")}
        className="font-mono min-h-32 resize-y"
      />
      <input
        value={tagsStr}
        onChange={(e) => setTagsStr(e.target.value)}
        placeholder={t("context.taskContext.tagsLabel")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <input
        value={filePath}
        onChange={(e) => setFilePath(e.target.value)}
        placeholder={t("context.taskContext.filePathLabel")}
        className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm"
      />
      <div className="flex items-center gap-2 justify-end">
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

function DocCard({
  doc,
  onEdit,
  onDelete,
}: {
  doc: TaskContext;
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
        <span className="text-sm font-medium flex-1 truncate">{doc.title}</span>
        {doc.tags.length > 0 && (
          <div className="hidden sm:flex items-center gap-1">
            {doc.tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs px-1.5">
                {tag}
              </Badge>
            ))}
          </div>
        )}
        {doc.file_path && (
          <span className="hidden sm:flex items-center gap-0.5 text-xs text-muted-foreground font-mono">
            <FolderOpen className="h-3 w-3" />
            {doc.file_path}
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
          {/* Show tags and file_path on mobile when expanded */}
          <div className="flex flex-wrap gap-1 mt-2 sm:hidden">
            {doc.tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs px-1.5">
                <Tag className="h-2.5 w-2.5 mr-0.5" />
                {tag}
              </Badge>
            ))}
            {doc.file_path && (
              <span className="flex items-center gap-0.5 text-xs text-muted-foreground font-mono">
                <FolderOpen className="h-3 w-3" />
                {doc.file_path}
              </span>
            )}
          </div>
          <pre className="text-xs font-mono whitespace-pre-wrap break-words mt-2 text-foreground/90 leading-relaxed">
            {doc.content}
          </pre>
        </div>
      )}
    </div>
  );
}

export function TaskContextEditor({ projectId }: Props) {
  const { t } = useTranslation("devflow");
  const { docs, loading, createDoc, updateDoc, deleteDoc } = useTaskContext(projectId);
  const showSkeleton = useDeferredLoading(loading && docs.length === 0);
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  return (
    <section>
      <h2 className="text-sm font-semibold mb-2">{t("context.taskContext.title")}</h2>

      {showSkeleton ? (
        <TableSkeleton rows={2} />
      ) : (
        <div className="space-y-2">
          {docs.length === 0 && !adding && (
            <EmptyState
              icon={Plus}
              title={t("context.taskContext.noDocuments")}
              description={t("context.taskContext.noDocumentsDesc")}
            />
          )}
          {docs.map((doc) =>
            editingId === doc.id ? (
              <DocForm
                key={doc.id}
                initial={{
                  title: doc.title,
                  content: doc.content,
                  tags: doc.tags,
                  file_path: doc.file_path ?? undefined,
                }}
                onSubmit={async (data) => {
                  await updateDoc(doc.id, data);
                  setEditingId(null);
                }}
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <DocCard
                key={doc.id}
                doc={doc}
                onEdit={() => { setEditingId(doc.id); setAdding(false); }}
                onDelete={() => deleteDoc(doc.id)}
              />
            ),
          )}
          {adding && (
            <DocForm
              onSubmit={async (data) => {
                await createDoc(data);
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
              <Plus className="h-3.5 w-3.5" /> {t("context.taskContext.add")}
            </Button>
          )}
        </div>
      )}
    </section>
  );
}
