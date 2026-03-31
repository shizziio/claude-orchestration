import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { CreateGitCredentialInput } from "@/types/devflow";

interface Props {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateGitCredentialInput) => Promise<void>;
}

export function GitCredentialFormDialog({ open, onClose, onSubmit }: Props) {
  const { t } = useTranslation("devflow");
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState<CreateGitCredentialInput>({
    label: "",
    provider: "github",
    auth_type: "token",
    host: "",
    token: "",
    private_key: "",
  });

  const set = (k: keyof CreateGitCredentialInput, v: string) =>
    setForm((f) => ({ ...f, [k]: v }));

  const handleSubmit = async () => {
    setLoading(true);
    try {
      await onSubmit({
        label: form.label,
        provider: form.provider,
        auth_type: form.auth_type,
        host: form.host || undefined,
        token: form.auth_type === "token" ? form.token || undefined : undefined,
        private_key: form.auth_type === "ssh_key" ? form.private_key || undefined : undefined,
      });
      setForm({ label: "", provider: "github", auth_type: "token", host: "", token: "", private_key: "" });
    } finally {
      setLoading(false);
    }
  };

  const isValid = form.label && form.provider && form.auth_type &&
    (form.auth_type === "token" ? !!form.token : !!form.private_key);

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md max-sm:inset-0 max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none flex flex-col max-h-[90dvh]">
        <DialogHeader className="shrink-0">
          <DialogTitle>{t("form.newCredential")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2 overflow-y-auto flex-1 pr-1">
          <div className="space-y-1">
            <Label>{t("form.label")}</Label>
            <Input value={form.label} onChange={(e) => set("label", e.target.value)} className="text-base md:text-sm" placeholder="My GitHub Token" />
          </div>
          <div className="space-y-1">
            <Label>{t("form.provider")}</Label>
            <select
              value={form.provider}
              onChange={(e) => set("provider", e.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2 text-base md:text-sm"
            >
              <option value="github">GitHub</option>
              <option value="gitlab">GitLab</option>
              <option value="bitbucket">Bitbucket</option>
              <option value="custom">{t("form.customProvider")}</option>
            </select>
          </div>
          {form.provider === "custom" && (
            <div className="space-y-1">
              <Label>{t("form.host")}</Label>
              <Input
                value={form.host}
                onChange={(e) => set("host", e.target.value)}
                placeholder="git.example.com"
                className="text-base md:text-sm font-mono"
              />
            </div>
          )}
          <div className="space-y-1">
            <Label>{t("form.authType")}</Label>
            <select
              value={form.auth_type}
              onChange={(e) => set("auth_type", e.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2 text-base md:text-sm"
            >
              <option value="token">{t("form.authTypeToken")}</option>
              <option value="ssh_key">{t("form.authTypeSSH")}</option>
            </select>
          </div>
          {form.auth_type === "token" && (
            <div className="space-y-1">
              <Label>{t("form.token")}</Label>
              <Input
                type="password"
                value={form.token}
                onChange={(e) => set("token", e.target.value)}
                placeholder="ghp_..."
                className="text-base md:text-sm font-mono"
              />
            </div>
          )}
          {form.auth_type === "ssh_key" && (
            <div className="space-y-1">
              <Label>{t("form.privateKey")}</Label>
              <Textarea
                value={form.private_key}
                onChange={(e) => set("private_key", e.target.value)}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className="text-base md:text-sm font-mono h-32 resize-none"
              />
            </div>
          )}
        </div>
        <DialogFooter className="shrink-0 pt-2">
          <Button variant="outline" onClick={onClose} disabled={loading}>{t("form.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={loading || !isValid}>
            {loading ? t("form.saving") : t("form.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
