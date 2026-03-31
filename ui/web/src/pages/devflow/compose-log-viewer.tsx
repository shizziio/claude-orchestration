import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Terminal, X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useHttp } from "@/hooks/use-ws";

interface Props {
  projectId: string;
  envId: string;
  envName: string;
  onClose: () => void;
}

export function ComposeLogViewer({ projectId, envId, envName, onClose }: Props) {
  const { t } = useTranslation("devflow");
  const http = useHttp();
  const [log, setLog] = useState("");
  const [connected, setConnected] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const controller = new AbortController();

    async function streamLogs() {
      try {
        const res = await http.streamFetch(
          `/v1/devflow/projects/${projectId}/environments/${envId}/logs?tail=200`,
          controller.signal,
        );
        setConnected(true);

        const reader = res.body?.getReader();
        if (!reader) return;

        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              const data = line.slice(6);
              if (data) {
                setLog((prev) => prev + data + "\n");
              }
            }
          }
        }
      } catch {
        // Aborted or stream ended
      } finally {
        setConnected(false);
      }
    }

    streamLogs();
    return () => controller.abort();
  }, [http, projectId, envId]);

  // Auto-scroll to bottom when log changes
  useEffect(() => {
    if (preRef.current) {
      const el = preRef.current.parentElement;
      if (el) {
        const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 100;
        if (isNearBottom) {
          el.scrollTop = el.scrollHeight;
        }
      }
    }
  }, [log]);

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background/80 backdrop-blur-sm">
      <div className="flex flex-col h-full max-w-4xl mx-auto w-full shadow-2xl bg-background border-x">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b shrink-0">
          <Terminal className="h-4 w-4 text-muted-foreground" />
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">
              {t("env.logs")} — {envName}
            </p>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              {connected && <Loader2 className="h-3 w-3 animate-spin" />}
              {connected ? "streaming..." : "disconnected"}
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} className="h-7 w-7 p-0 shrink-0">
            <X className="h-4 w-4" />
          </Button>
        </div>
        {/* Log output */}
        <div className="flex-1 overflow-y-auto p-4 overscroll-contain">
          {log ? (
            <pre
              ref={preRef}
              className="text-xs font-mono whitespace-pre-wrap break-words text-foreground/90 leading-relaxed"
            >
              {log}
            </pre>
          ) : (
            <p className="text-sm text-muted-foreground italic">
              {connected ? "Waiting for log output..." : "No logs available."}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
