import { useState, useRef, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Send, Loader2, CheckCircle, XCircle, Clock, Bot, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { formatRelativeTime } from "@/lib/format";
import { useRuns, useRunLog } from "./hooks/use-devflow";
import type { DevflowRun } from "@/types/devflow";

interface Props {
  projectId: string;
  defaultBranch: string;
}

interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  runId?: string;
  timestamp: Date;
}

function RunStatusInline({ status }: { status: string }) {
  if (status === "running") return <Loader2 className="inline h-3 w-3 animate-spin text-blue-500" />;
  if (status === "completed") return <CheckCircle className="inline h-3 w-3 text-green-500" />;
  if (status === "failed") return <XCircle className="inline h-3 w-3 text-red-500" />;
  return <Clock className="inline h-3 w-3 text-muted-foreground" />;
}

function RunResultBubble({ projectId, run }: { projectId: string; run: DevflowRun }) {
  const [showLog, setShowLog] = useState(false);
  const { log } = useRunLog(projectId, run.id, showLog ? run.status : "");

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <RunStatusInline status={run.status} />
        <Badge variant={run.status === "completed" ? "default" : run.status === "failed" ? "destructive" : "secondary"} className="text-xs">
          {run.status}
        </Badge>
        {run.cost_usd != null && run.cost_usd > 0 && (
          <span className="text-xs text-muted-foreground">${run.cost_usd.toFixed(2)}</span>
        )}
        {run.duration_ms != null && (
          <span className="text-xs text-muted-foreground">{Math.round(run.duration_ms / 1000)}s</span>
        )}
      </div>
      {run.result_summary && (
        <p className="text-sm text-foreground/80">{run.result_summary}</p>
      )}
      {run.error_message && (
        <p className="text-sm text-red-500">{run.error_message}</p>
      )}
      <button
        onClick={() => setShowLog(!showLog)}
        className="text-xs text-blue-500 hover:underline"
      >
        {showLog ? "Hide log" : "Show log"}
      </button>
      {showLog && log && (
        <pre className="mt-1 max-h-48 overflow-y-auto rounded bg-muted p-2 text-xs font-mono whitespace-pre-wrap">
          {log}
        </pre>
      )}
    </div>
  );
}

export function QuickChat({ projectId, defaultBranch }: Props) {
  const { t } = useTranslation("devflow");
  const { runs, createRun, startRun } = useRuns(projectId);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll on new messages
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, runs]);

  // Map run IDs to latest run data
  const runMap = new Map(runs.map((r) => [r.id, r]));

  const handleSend = async () => {
    const task = input.trim();
    if (!task || sending) return;

    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      role: "user",
      content: task,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setSending(true);

    try {
      const run = await createRun({ task_description: task, branch: defaultBranch });
      await startRun(run.id);

      const assistantMsg: ChatMessage = {
        id: crypto.randomUUID(),
        role: "assistant",
        content: "",
        runId: run.id,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch {
      const errMsg: ChatMessage = {
        id: crypto.randomUUID(),
        role: "assistant",
        content: t("chat.failedToStart"),
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, errMsg]);
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  };

  return (
    <section className="rounded-lg border bg-card">
      <div className="flex items-center gap-2 border-b px-4 py-2.5">
        <Bot className="h-4 w-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold">{t("chat.title")}</h2>
        <span className="text-xs text-muted-foreground ml-auto">{t("chat.hint")}</span>
      </div>

      {/* Messages */}
      <div className="max-h-96 min-h-[120px] overflow-y-auto px-4 py-3 space-y-3">
        {messages.length === 0 && (
          <p className="text-sm text-muted-foreground italic text-center py-4">
            {t("chat.empty")}
          </p>
        )}
        {messages.map((msg) => (
          <div key={msg.id} className={`flex gap-2 ${msg.role === "user" ? "justify-end" : ""}`}>
            {msg.role === "assistant" && (
              <Bot className="h-5 w-5 mt-0.5 shrink-0 text-muted-foreground" />
            )}
            <div
              className={`rounded-lg px-3 py-2 max-w-[85%] ${
                msg.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted"
              }`}
            >
              {msg.role === "user" ? (
                <p className="text-sm">{msg.content}</p>
              ) : msg.runId && runMap.has(msg.runId) ? (
                <RunResultBubble projectId={projectId} run={runMap.get(msg.runId)!} />
              ) : (
                <p className="text-sm">{msg.content || t("chat.processing")}</p>
              )}
              <p className="text-[10px] opacity-60 mt-1">
                {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
              </p>
            </div>
            {msg.role === "user" && (
              <User className="h-5 w-5 mt-0.5 shrink-0 text-muted-foreground" />
            )}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="flex items-center gap-2 border-t px-4 py-2.5">
        <input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleSend()}
          placeholder={t("chat.placeholder")}
          disabled={sending}
          className="flex-1 rounded-md border bg-background px-3 py-2 text-base md:text-sm focus:outline-none focus:ring-1 focus:ring-ring"
        />
        <Button
          size="sm"
          onClick={handleSend}
          disabled={!input.trim() || sending}
          className="h-9 w-9 p-0 shrink-0"
        >
          {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
        </Button>
      </div>
    </section>
  );
}
