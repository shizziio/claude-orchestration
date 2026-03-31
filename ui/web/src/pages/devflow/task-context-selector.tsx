import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { useTaskContext } from "./hooks/use-context";

interface Props {
  projectId: string;
  selectedIds: string[];
  onChange: (ids: string[]) => void;
}

export function TaskContextSelector({ projectId, selectedIds, onChange }: Props) {
  const { t } = useTranslation("devflow");
  const { docs, loading } = useTaskContext(projectId);

  const toggle = (id: string) => {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((x) => x !== id));
    } else {
      onChange([...selectedIds, id]);
    }
  };

  if (loading) {
    return <p className="text-xs text-muted-foreground">{t("context.loadingDocs")}</p>;
  }

  if (docs.length === 0) {
    return <p className="text-xs text-muted-foreground">{t("context.noTaskDocs")}</p>;
  }

  return (
    <div className="space-y-1">
      <label className="text-sm text-muted-foreground">{t("context.taskContextLabel")}</label>
      <div className="max-h-40 overflow-y-auto rounded-md border p-2 space-y-1 overscroll-contain">
        {docs.map((doc) => (
          <label
            key={doc.id}
            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted/40 cursor-pointer"
          >
            <input
              type="checkbox"
              checked={selectedIds.includes(doc.id)}
              onChange={() => toggle(doc.id)}
              className="rounded border-input"
            />
            <span className="text-sm flex-1 truncate">{doc.title}</span>
            {doc.tags?.length > 0 && (
              <div className="flex gap-1 shrink-0">
                {doc.tags.slice(0, 2).map((tag) => (
                  <Badge key={tag} variant="outline" className="text-xs px-1.5">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
          </label>
        ))}
      </div>
    </div>
  );
}
