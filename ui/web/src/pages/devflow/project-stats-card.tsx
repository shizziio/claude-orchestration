import { useTranslation } from "react-i18next";
import { useProjectStats } from "./hooks/use-devflow";

function formatDuration(ms: number): string {
  const totalSeconds = Math.round(ms / 1000);
  if (totalSeconds < 60) return `${totalSeconds}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
}

function formatCost(usd: number): string {
  return `$${usd.toFixed(2)}`;
}

function successRateColor(rate: number): string {
  if (rate >= 0.8) return "text-green-600";
  if (rate >= 0.5) return "text-yellow-600";
  return "text-red-600";
}

interface Props {
  projectId: string;
}

export function ProjectStatsCard({ projectId }: Props) {
  const { t } = useTranslation("devflow");
  const { stats, loading } = useProjectStats(projectId);

  if (loading || !stats || stats.total_runs === 0) return null;

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div className="rounded-lg border p-3">
        <p className="text-xs text-muted-foreground">{t("stats.totalRuns")}</p>
        <p className="text-xl font-semibold mt-1">{stats.total_runs}</p>
      </div>
      <div className="rounded-lg border p-3">
        <p className="text-xs text-muted-foreground">{t("stats.successRate")}</p>
        <p className={`text-xl font-semibold mt-1 ${successRateColor(stats.success_rate)}`}>
          {Math.round(stats.success_rate * 100)}%
        </p>
      </div>
      <div className="rounded-lg border p-3">
        <p className="text-xs text-muted-foreground">{t("stats.avgDuration")}</p>
        <p className="text-xl font-semibold mt-1">
          {stats.avg_duration_ms > 0 ? formatDuration(stats.avg_duration_ms) : "\u2014"}
        </p>
      </div>
      <div className="rounded-lg border p-3">
        <p className="text-xs text-muted-foreground">{t("stats.totalCost")}</p>
        <p className="text-xl font-semibold mt-1">
          {stats.total_cost_usd > 0 ? formatCost(stats.total_cost_usd) : "\u2014"}
        </p>
      </div>
    </div>
  );
}
