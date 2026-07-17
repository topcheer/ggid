"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck, Users, AlertTriangle, ClipboardCheck, Loader2,
  AlertCircle, X, Clock, UserX, TrendingUp,
} from "lucide-react";

interface IGAMetrics {
  open_campaigns: number;
  pending_reviews: number;
  overdue_reviews: number;
  sod_violations: { critical: number; high: number; medium: number };
  orphaned_accounts: number;
  dormant_accounts: number;
  cert_completion_rate: number;
  avg_review_time_hours: number;
  recent_campaigns: { id: string; name: string; status: string; completion: number }[];
}

export default function IdentityGovernancePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [metrics, setMetrics] = useState<IGAMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setMetrics(await apiFetch<IGAMetrics>("/api/v1/audit/iga/metrics").catch(() => null)); }
      catch { setError("Failed to load IGA metrics"); }
      finally { setLoading(false); }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;
  if (error || !metrics) return <div className={cardCls}><div className="py-12 text-center"><AlertCircle className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{error ?? "No IGA data available."}</p></div></div>;

  const sodTotal = metrics.sod_violations.critical + metrics.sod_violations.high + metrics.sod_violations.medium;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-indigo-600" /> {t("backend.identityGovernance.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Unified dashboard for access reviews, SoD violations, and account lifecycle.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Top metrics */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className={cardCls}><div className="flex items-center gap-2"><ClipboardCheck className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.openCampaigns")}</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{metrics.open_campaigns}</p></div>
        <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.pendingReviews")}</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{metrics.pending_reviews}</p>{metrics.overdue_reviews > 0 && <p className="mt-1 text-xs text-red-500">{metrics.overdue_reviews} overdue</p>}</div>
        <div className={cardCls}><div className="flex items-center gap-2"><AlertTriangle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.sodViolations")}</span></div><p className="mt-2 text-2xl font-bold text-red-600">{sodTotal}</p>{metrics.sod_violations.critical > 0 && <p className="mt-1 text-xs text-red-500">{metrics.sod_violations.critical} critical</p>}</div>
        <div className={cardCls}><div className="flex items-center gap-2"><UserX className="h-4 w-4 text-gray-400" /><span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.orphanedAccounts")}</span></div><p className="mt-2 text-2xl font-bold text-gray-600">{metrics.orphaned_accounts}</p></div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Certification progress */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><TrendingUp className="h-4 w-4" /> Certification Progress</h3>
          <div className="space-y-3">
            <div><div className="flex items-center justify-between text-sm"><span className="text-gray-500">{t("backend.identityGovernance.completionRate")}</span><span className="font-bold text-green-600">{metrics.cert_completion_rate}%</span></div><div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-green-500" style={{ width: `${metrics.cert_completion_rate}%` }} /></div></div>
            <div className="flex items-center justify-between text-sm"><span className="text-gray-500">{t("backend.identityGovernance.avgReviewTime")}</span><span className="font-bold text-indigo-600">{metrics.avg_review_time_hours}h</span></div>
            <div className="flex items-center justify-between text-sm"><span className="text-gray-500">{t("backend.identityGovernance.dormantAccounts")}</span><span className="font-bold text-gray-600">{metrics.dormant_accounts}</span></div>
          </div>
        </div>

        {/* SoD breakdown */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><AlertTriangle className="h-4 w-4" /> SoD Violation Breakdown</h3>
          <div className="space-y-2">
            {["critical", "high", "medium"].map((sev) => {
              const count = metrics.sod_violations[sev as keyof typeof metrics.sod_violations];
              const colors = { critical: "bg-red-500", high: "bg-orange-500", medium: "bg-yellow-500" };
              const max = Math.max(sodTotal, 1);
              return (
                <div key={sev}>
                  <div className="flex items-center justify-between text-sm"><span className="capitalize text-gray-500">{sev}</span><span className="font-bold text-gray-700 dark:text-gray-300">{count}</span></div>
                  <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={`h-full rounded-full ${colors[sev as keyof typeof colors]}`} style={{ width: `${(count / max) * 100}%` }} /></div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Recent campaigns */}
      {metrics.recent_campaigns.length > 0 && (
        <div>
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.identityGovernance.recentCampaigns")}</h2>
          <div className="space-y-2">
            {metrics.recent_campaigns.map((c) => (
              <div key={c.id} className={`${cardCls} flex items-center justify-between py-3`}>
                <div><span className="font-medium text-gray-800 dark:text-gray-200">{c.name}</span></div>
                <div className="flex items-center gap-3">
                  <div className="h-1.5 w-20 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-500" style={{ width: `${c.completion}%` }} /></div>
                  <span className="text-xs text-gray-400">{c.completion}%</span>
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${c.status === "completed" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"}`}>{c.status}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
