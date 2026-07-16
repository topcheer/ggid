"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldAlert, Download, AlertTriangle, Loader2, X, AlertCircle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SoDViolation {
  id: string;
  user_id: string;
  user_name: string;
  conflicting_roles: string[];
  rule_description: string;
  severity: "critical" | "high" | "medium";
  detected_at: string;
}

const SEV_CONFIG = {
  critical: { icon: ShieldAlert, color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
  high: { icon: AlertTriangle, color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" },
  medium: { icon: AlertTriangle, color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
};

export default function SoDViolationsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [violations, setViolations] = useState<SoDViolation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [severityFilter, setSeverityFilter] = useState<string>("all");
  const [exporting, setExporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ violations?: SoDViolation[]; items?: SoDViolation[] }>("/api/v1/policy/sod/violations").catch(() => null);
      setViolations(data?.violations ?? data?.items ?? []);
    } catch {
      setError("Failed to load SoD violations");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const filtered = severityFilter === "all" ? violations : violations.filter((v) => v.severity === severityFilter);
  const counts = {
    critical: violations.filter((v) => v.severity === "critical").length,
    high: violations.filter((v) => v.severity === "high").length,
    medium: violations.filter((v) => v.severity === "medium").length,
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      const resp = await apiFetch<Response>(`/api/v1/policy/sod/violations/export?format=csv`).catch(() => null);
      if (resp && resp instanceof Response && resp.ok) {
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `sod-violations-${new Date().toISOString().split("T")[0]}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      }
    } catch {
      setError("Export failed");
    } finally {
      setExporting(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldAlert className="h-6 w-6 text-red-600" /> SoD Violations
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Separation of Duties conflicts detected across user role assignments.</p>
        </div>
        <button onClick={handleExport} disabled={exporting || violations.length === 0} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
          {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />} Export CSV
        </button>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Severity summary */}
      <div className="grid grid-cols-3 gap-4">
        {(["critical", "high", "medium"] as const).map((sev) => {
          const cfg = SEV_CONFIG[sev];
          const Icon = cfg.icon;
          return (
            <button key={sev} onClick={() => setSeverityFilter(severityFilter === sev ? "all" : sev)} className={`${cardCls} ${severityFilter === sev ? "ring-2 ring-indigo-400" : ""}`}>
              <div className="flex items-center gap-2">
                <div className={`rounded-lg ${cfg.bg} p-1.5`}><Icon className={`h-4 w-4 ${cfg.color}`} /></div>
                <span className="text-xs font-semibold uppercase text-gray-500">{sev}</span>
              </div>
              <p className={`mt-2 text-2xl font-bold ${cfg.color}`}>{counts[sev]}</p>
            </button>
          );
        })}
      </div>

      {/* Violations table */}
      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : filtered.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><ShieldAlert className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{severityFilter === "all" ? "No SoD violations detected." : `No ${severityFilter} severity violations.`}</p></div></div>
      ) : (
        <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800"><tr className="text-left text-xs font-semibold uppercase text-gray-500">
              <th className="px-4 py-3">User</th><th className="px-4 py-3">Conflicting Roles</th><th className="px-4 py-3">Rule</th><th className="px-4 py-3">Severity</th><th className="px-4 py-3">Detected</th>
            </tr></thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {filtered.map((v) => {
                const cfg = SEV_CONFIG[v.severity];
                const SevIcon = cfg.icon;
                return (
                  <tr key={v.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3 font-medium text-gray-800 dark:text-gray-200">{v.user_name}</td>
                    <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{v.conflicting_roles.map((r) => <span key={r} className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-400">{r}</span>)}</div></td>
                    <td className="px-4 py-3 text-gray-500">{v.rule_description}</td>
                    <td className="px-4 py-3"><span className={`flex items-center gap-1 rounded-full ${cfg.bg} px-2 py-0.5 text-xs font-medium ${cfg.color}`}><SevIcon className="h-3 w-3" />{v.severity}</span></td>
                    <td className="px-4 py-3 text-gray-400">{new Date(v.detected_at).toLocaleString()}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Mobile cards */}
      {!loading && filtered.length > 0 && (
        <div className="space-y-3 md:hidden">
          {filtered.map((v) => {
            const cfg = SEV_CONFIG[v.severity];
            return (
              <div key={v.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <span className="font-medium text-gray-800 dark:text-gray-200">{v.user_name}</span>
                  <span className={`rounded-full ${cfg.bg} px-2 py-0.5 text-xs font-medium ${cfg.color}`}>{v.severity}</span>
                </div>
                <div className="mt-1 flex flex-wrap gap-1">{v.conflicting_roles.map((r) => <span key={r} className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-400">{r}</span>)}</div>
                <p className="mt-1 text-xs text-gray-400">{v.rule_description}</p>
                <p className="mt-1 text-xs text-gray-400">Detected: {new Date(v.detected_at).toLocaleString()}</p>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
