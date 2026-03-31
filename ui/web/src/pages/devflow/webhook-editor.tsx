import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Plus, Pencil, Trash2, Loader2, Webhook, Copy, Check,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useWebhooks } from "./hooks/use-devflow";
import type { DevflowWebhook, CreateWebhookInput } from "@/types/devflow";

interface Props {
  projectId: string;
}

function WebhookForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Partial<CreateWebhookInput>;
  onSubmit: (data: CreateWebhookInput) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation("devflow");
  const [eventType, setEventType] = useState(initial?.event_type ?? "push");
  const [branchFilter, setBranchFilter] = useState(initial?.branch_filter ?? "");
  const [taskTemplate, setTaskTemplate] = useState(initial?.task_template ?? "");
  const [secret, setSecret] = useState(initial?.secret ?? "");
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    if (!taskTemplate.trim()) return;
    setSaving(true);
    try {
      await onSubmit({
        event_type: eventType,
        branch_filter: branchFilter || undefined,
        task_template: taskTemplate,
        secret: secret || undefined,
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <div>
        <label className="text-xs text-muted-foreground">{t("webhooks.eventType")}</label>
        <select
          value={eventType}
          onChange={(e) => setEventType(e.target.value)}
          className="w-full rounded-md border bg-background px-3 py-2 text-base md:text-sm mt-1"
        >
          <option value="push">{t("webhooks.push")}</option>
          <option value="pull_request">{t("webhooks.pullRequest")}</option>
        </select>
      </div>
      <div>
        <label className="text-xs text-muted-foreground">{t("webhooks.branchFilter")}</label>
        <input
          value={branchFilter}
          onChange={(e) => setBranchFilter(e.target.value)}
          placeholder="main|develop|feature/.*"
          className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm font-mono mt-1"
        />
      </div>
      <div>
        <label className="text-xs text-muted-foreground">{t("webhooks.taskTemplate")}</label>
        <Textarea
          value={taskTemplate}
          onChange={(e) => setTaskTemplate(e.target.value)}
          placeholder={t("webhooks.taskTemplateHint")}
          className="font-mono min-h-24 resize-y mt-1 text-base md:text-sm"
        />
        <p className="text-xs text-muted-foreground mt-1">{t("webhooks.taskTemplateHint")}</p>
      </div>
      <div>
        <label className="text-xs text-muted-foreground">{t("webhooks.secret")}</label>
        <input
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          className="w-full rounded-md border bg-background px-3 py-1.5 text-base md:text-sm font-mono mt-1"
        />
      </div>
      <div className="flex gap-2 justify-end">
        <Button variant="outline" size="sm" onClick={onCancel} disabled={saving}>
          {t("form.cancel")}
        </Button>
        <Button size="sm" onClick={handleSubmit} disabled={saving || !taskTemplate.trim()}>
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : null}
          {t("form.save")}
        </Button>
      </div>
    </div>
  );
}

function WebhookCard({
  webhook,
  onEdit,
  onDelete,
  onToggle,
}: {
  webhook: DevflowWebhook;
  onEdit: () => void;
  onDelete: () => void;
  onToggle: () => void;
}) {
  const { t } = useTranslation("devflow");

  return (
    <div className="rounded-lg border px-3 py-2">
      <div className="flex items-center gap-2">
        <Badge variant="outline">
          {webhook.event_type === "push" ? t("webhooks.push") : t("webhooks.pullRequest")}
        </Badge>
        {webhook.branch_filter && (
          <span className="text-xs font-mono text-muted-foreground truncate max-w-32">
            {webhook.branch_filter}
          </span>
        )}
        <span className="flex-1 text-xs text-muted-foreground truncate hidden sm:inline">
          {webhook.task_template.slice(0, 80)}{webhook.task_template.length > 80 ? "..." : ""}
        </span>
        <button
          type="button"
          onClick={onToggle}
          className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
            webhook.enabled ? "bg-primary" : "bg-muted"
          }`}
          title={t("webhooks.enabled")}
        >
          <span
            className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-background shadow ring-0 transition-transform ${
              webhook.enabled ? "translate-x-4" : "translate-x-0"
            }`}
          />
        </button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={onEdit}
        >
          <Pencil className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      <p className="text-xs text-muted-foreground mt-1 truncate sm:hidden">
        {webhook.task_template.slice(0, 60)}{webhook.task_template.length > 60 ? "..." : ""}
      </p>
    </div>
  );
}

function IncomingUrl({ projectId }: { projectId: string }) {
  const { t } = useTranslation("devflow");
  const [copied, setCopied] = useState(false);
  const url = `${window.location.origin}/v1/devflow/webhooks/incoming/${projectId}`;

  const copy = () => {
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-lg border p-3 space-y-1">
      <label className="text-xs font-medium text-muted-foreground">{t("webhooks.incomingUrl")}</label>
      <div className="flex items-center gap-2">
        <code className="flex-1 text-xs font-mono bg-muted px-2 py-1.5 rounded truncate">
          {url}
        </code>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0 shrink-0" onClick={copy}>
          {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
        </Button>
      </div>
    </div>
  );
}

export function WebhookEditor({ projectId }: Props) {
  const { t } = useTranslation("devflow");
  const { webhooks, loading, createWebhook, updateWebhook, deleteWebhook } = useWebhooks(projectId);
  const showSkeleton = useDeferredLoading(loading && webhooks.length === 0);
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  return (
    <section>
      <h2 className="text-sm font-semibold mb-2">{t("webhooks.title")}</h2>

      {showSkeleton ? (
        <TableSkeleton rows={2} />
      ) : (
        <div className="space-y-2">
          <IncomingUrl projectId={projectId} />

          {webhooks.length === 0 && !adding && (
            <EmptyState
              icon={Webhook}
              title={t("webhooks.noWebhooks")}
              description={t("webhooks.noWebhooksDesc")}
            />
          )}
          {webhooks.map((wh) =>
            editingId === wh.id ? (
              <WebhookForm
                key={wh.id}
                initial={{
                  event_type: wh.event_type,
                  branch_filter: wh.branch_filter ?? undefined,
                  task_template: wh.task_template,
                }}
                onSubmit={async (data) => {
                  await updateWebhook(wh.id, data);
                  setEditingId(null);
                }}
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <WebhookCard
                key={wh.id}
                webhook={wh}
                onEdit={() => { setEditingId(wh.id); setAdding(false); }}
                onDelete={() => deleteWebhook(wh.id)}
                onToggle={() => updateWebhook(wh.id, { enabled: !wh.enabled })}
              />
            ),
          )}
          {adding && (
            <WebhookForm
              onSubmit={async (data) => {
                await createWebhook(data);
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
              <Plus className="h-3.5 w-3.5" /> {t("webhooks.add")}
            </Button>
          )}
        </div>
      )}
    </section>
  );
}
