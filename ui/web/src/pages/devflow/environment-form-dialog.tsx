import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

interface CreateEnvInput {
  name: string;
  slug: string;
  branch?: string;
  docker_compose_override?: string;
}

interface Props {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateEnvInput) => Promise<void>;
}

export function EnvironmentFormDialog({ open, onClose, onSubmit }: Props) {
  const { t } = useTranslation("devflow");
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState<CreateEnvInput>({
    name: "",
    slug: "",
    branch: "",
    docker_compose_override: "",
  });

  const set = (k: keyof CreateEnvInput, v: string) =>
    setForm((f) => ({ ...f, [k]: v }));

  // Auto-generate slug from name
  const handleNameChange = (v: string) => {
    set("name", v);
    set("slug", v.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""));
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      await onSubmit({
        name: form.name,
        slug: form.slug,
        branch: form.branch || undefined,
        docker_compose_override: form.docker_compose_override || undefined,
      });
      setForm({ name: "", slug: "", branch: "", docker_compose_override: "" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md max-sm:inset-0 max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none">
        <DialogHeader>
          <DialogTitle>{t("env.newEnv")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="space-y-1">
            <Label>{t("env.nameLabel")}</Label>
            <Input value={form.name} onChange={(e) => handleNameChange(e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label>{t("env.slugLabel")}</Label>
            <Input value={form.slug} onChange={(e) => set("slug", e.target.value)} className="text-base md:text-sm font-mono" />
          </div>
          <div className="space-y-1">
            <Label>{t("env.branchLabel")}</Label>
            <Input value={form.branch} onChange={(e) => set("branch", e.target.value)} className="text-base md:text-sm" />
          </div>
          <div className="space-y-1">
            <Label>{t("env.composeOverrideLabel")}</Label>
            <Textarea
              value={form.docker_compose_override}
              onChange={(e) => set("docker_compose_override", e.target.value)}
              className="font-mono min-h-24 resize-y text-base md:text-sm"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>{t("form.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={loading || !form.name || !form.slug}>
            {loading ? t("form.saving") : t("form.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
