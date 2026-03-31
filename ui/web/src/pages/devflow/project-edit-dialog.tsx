import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Project, GitCredential } from "@/types/devflow";

interface EditForm {
  name: string;
  description: string;
  repo_url: string;
  default_branch: string;
  git_credential_id: string;
}

interface Props {
  open: boolean;
  project: Project;
  credentials: GitCredential[];
  onClose: () => void;
  onSubmit: (data: Partial<EditForm>) => Promise<void>;
}

export function ProjectEditDialog({ open, project, credentials, onClose, onSubmit }: Props) {
  const { t } = useTranslation("devflow");
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState<EditForm>({
    name: "",
    description: "",
    repo_url: "",
    default_branch: "main",
    git_credential_id: "",
  });

  // Sync form with project whenever dialog opens
  useEffect(() => {
    if (open) {
      setForm({
        name: project.name,
        description: project.description ?? "",
        repo_url: project.repo_url ?? "",
        default_branch: project.default_branch,
        git_credential_id: project.git_credential_id ?? "",
      });
    }
  }, [open, project]);

  const set = (k: keyof EditForm, v: string) =>
    setForm((f) => ({ ...f, [k]: v }));

  const handleSubmit = async () => {
    setLoading(true);
    try {
      await onSubmit({
        name: form.name || undefined,
        description: form.description || undefined,
        repo_url: form.repo_url || undefined,
        default_branch: form.default_branch || undefined,
        git_credential_id: form.git_credential_id || undefined,
      });
      onClose();
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md max-sm:inset-0 max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none">
        <DialogHeader>
          <DialogTitle>{t("form.editProject")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="space-y-1">
            <Label>{t("form.name")}</Label>
            <Input value={form.name} onChange={(e) => set("name", e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label>{t("form.description")}</Label>
            <Input value={form.description} onChange={(e) => set("description", e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label>{t("form.repoUrl")}</Label>
            <Input
              value={form.repo_url}
              onChange={(e) => set("repo_url", e.target.value)}
              placeholder="https://github.com/org/repo"
              className="text-base md:text-sm font-mono"
            />
          </div>
          <div className="space-y-1">
            <Label>{t("form.defaultBranch")}</Label>
            <Input value={form.default_branch} onChange={(e) => set("default_branch", e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label>{t("form.gitCredential")}</Label>
            <select
              value={form.git_credential_id}
              onChange={(e) => set("git_credential_id", e.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2 text-base md:text-sm"
            >
              <option value="">{t("form.noCredential")}</option>
              {credentials.map((c) => (
                <option key={c.id} value={c.id}>{c.label} ({c.provider})</option>
              ))}
            </select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>{t("form.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={loading || !form.name}>
            {loading ? t("form.saving") : t("form.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
