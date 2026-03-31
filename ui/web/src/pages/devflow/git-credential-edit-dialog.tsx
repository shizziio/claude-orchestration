import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { GitCredential } from "@/types/devflow";

interface Props {
  open: boolean;
  credential: GitCredential;
  onClose: () => void;
  onSubmit: (data: { label?: string; host?: string; private_key?: string; token?: string }) => Promise<void>;
}

export function GitCredentialEditDialog({ open, credential, onClose, onSubmit }: Props) {
  const { t } = useTranslation("devflow");
  const [loading, setLoading] = useState(false);
  const [label, setLabel] = useState("");
  const [host, setHost] = useState("");
  const [secret, setSecret] = useState(""); // private_key or token — leave blank to keep existing

  useEffect(() => {
    if (open) {
      setLabel(credential.label);
      setHost(credential.host ?? "");
      setSecret("");
    }
  }, [open, credential]);

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const data: Parameters<typeof onSubmit>[0] = {};
      if (label && label !== credential.label) data.label = label;
      if (host !== (credential.host ?? "")) data.host = host;
      if (secret.trim()) {
        if (credential.auth_type === "ssh_key") {
          data.private_key = secret;
        } else {
          data.token = secret;
        }
      }
      await onSubmit(data);
      onClose();
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md max-sm:inset-0 max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none flex flex-col max-h-[90dvh]">
        <DialogHeader className="shrink-0">
          <DialogTitle>{t("form.editCredential")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2 overflow-y-auto flex-1 pr-1">
          <div className="space-y-1">
            <Label>{t("form.label")}</Label>
            <Input value={label} onChange={(e) => setLabel(e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label className="text-muted-foreground text-xs">{t("form.provider")}</Label>
            <p className="text-sm px-1">{credential.provider}</p>
          </div>
          {credential.host !== null && (
            <div className="space-y-1">
              <Label>{t("form.host")}</Label>
              <Input
                value={host}
                onChange={(e) => setHost(e.target.value)}
                className="text-base md:text-sm font-mono"
                placeholder="git.example.com"
              />
            </div>
          )}
          <div className="space-y-1">
            <Label>
              {credential.auth_type === "ssh_key" ? t("form.privateKey") : t("form.token")}
              <span className="ml-2 text-xs text-muted-foreground">({t("form.leaveBlankToKeep")})</span>
            </Label>
            {credential.auth_type === "ssh_key" ? (
              <Textarea
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className="text-base md:text-sm font-mono h-32 resize-none"
              />
            ) : (
              <Input
                type="password"
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
                placeholder={t("form.tokenPlaceholder")}
                className="text-base md:text-sm font-mono"
              />
            )}
          </div>
        </div>
        <DialogFooter className="shrink-0 pt-2">
          <Button variant="outline" onClick={onClose} disabled={loading}>{t("form.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={loading || !label}>
            {loading ? t("form.saving") : t("form.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
