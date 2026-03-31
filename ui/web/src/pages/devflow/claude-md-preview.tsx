import { useTranslation } from "react-i18next";
import { RefreshCw, Loader2, FileText, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useClaudeMdPreview } from "./hooks/use-context";

interface Props {
  projectId: string;
  onClose: () => void;
}

export function ClaudeMdPreview({ projectId, onClose }: Props) {
  const { t } = useTranslation("devflow");
  const { content, loading, refresh } = useClaudeMdPreview(projectId);

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background/80 backdrop-blur-sm">
      <div className="flex flex-col h-full max-w-4xl mx-auto w-full shadow-2xl bg-background border-x">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b shrink-0">
          <FileText className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium flex-1">{t("context.claudeMdTitle")}</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => refresh()}
            disabled={loading}
            className="h-7 w-7 p-0"
          >
            {loading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
          </Button>
          <Button variant="ghost" size="sm" onClick={onClose} className="h-7 w-7 p-0">
            <X className="h-4 w-4" />
          </Button>
        </div>
        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 overscroll-contain">
          {loading && !content ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : content ? (
            <pre className="text-xs font-mono whitespace-pre-wrap break-words text-foreground/90 leading-relaxed">
              {content}
            </pre>
          ) : (
            <p className="text-sm text-muted-foreground italic">{t("context.claudeMdEmpty")}</p>
          )}
        </div>
      </div>
    </div>
  );
}
