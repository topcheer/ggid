"use client";

import { useState, useCallback, useEffect } from "react";
import { Search, ShieldCheck, ShieldX, Shield, Clock, Loader2, AlertCircle, X, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

interface Permission {
  id: string;
  resource: string;
  action: string;
  source: "direct" | "inherited";
  via_group: string | null;
  last_used: string | null;
  unused_90d: boolean;
  over_privileged: boolean;
  recommendation: "keep" | "revoke" | "reduce";
}

interface CrossAnalysisRow {
  resource: string;
  granted_count: number;
  used_count: number;
  unused_count: number;
  utilization_pct: number;
  risk_level: "low" | "medium" | "high";
}

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

const recConfig: Record<string, { color: string; icon: typeof ShieldCheck; label: string }> = {
  keep: { color: "text-green-600", icon: ShieldCheck, label: "Keep" },
  revoke: { color: "text-red-600", icon: ShieldX, label: "Revoke" },
  reduce: { color: "text-yellow-600", icon: Shield, label: "Reduce" },
};

const riskColors: Record<string, string> = {
  high: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  medium: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  low: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
};

export default function EntitlementReviewPage() {
  const t = useTranslations();
  const [search, setSearch] = useState("");
  const [perms, setPerms] = useState<Permission[]>([]);
  const [crossAnalysis, setCrossAnalysis] = useState<CrossAnalysisRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [crossLoading, setCrossLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load cross-analysis data on mount
  const loadCrossAnalysis = useCallback(async () => {
    setCrossLoading(true);
    try {
      const res = await fetch("/api/v1/identity/entitlement-review/cross-analysis", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const data = await res.json();
        setCrossAnalysis(data.rows || data.cross_analysis || []);
      }
    } catch { /* endpoint not implemented yet */ }
    finally { setCrossLoading(false); }
  }, []);

  useEffect(() => { loadCrossAnalysis(); }, [loadCrossAnalysis]);

  const searchUser = useCallback(async () => {
    if (!search) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/identity/entitlement-review?user=${encodeURIComponent(search)}`, {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      if (res.ok) {
        const d = await res.json();
        setPerms(d.permissions || d || []);
      } else {
        setError("User not found or no entitlement data");
      }
    } catch {
      setError("Failed to fetch entitlement data");
    }
    finally { setLoading(false); }
  }, [search]);

  const unused = perms.filter((p) => p.unused_90d).length;
  const overPriv = perms.filter((p) => p.over_privileged).length;

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="w-6 h-6 text-blue-500" />
            {t("entitlementReview.title") || "CIEM Entitlement Review"}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Permission usage analytics, granted vs. used cross-analysis, and right-sizing recommendations.
          </p>
        </div>
        <button onClick={loadCrossAnalysis} disabled={crossLoading} aria-label="Refresh cross-analysis" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (crossLoading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {/* Error */}
      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Granted × Used Cross-Analysis */}
      <div className={cardCls}>
        <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Granted × Used Cross-Analysis</h2>
        {crossLoading ? (
          <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-blue-500" /></div>
        ) : crossAnalysis.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Resource</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Granted</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Used</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Unused</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Utilization</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Risk</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {crossAnalysis.map((row, i) => (
                  <tr key={`${row.resource}-${i}`} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3 font-mono text-xs">{row.resource}</td>
                    <td className="px-4 py-3 text-right font-medium">{row.granted_count}</td>
                    <td className="px-4 py-3 text-right text-green-600">{row.used_count}</td>
                    <td className="px-4 py-3 text-right text-red-600">{row.unused_count}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-20 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                          <div className={"h-full rounded-full " + (row.utilization_pct >= 70 ? "bg-green-500" : row.utilization_pct >= 30 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${row.utilization_pct}%` }} />
                        </div>
                        <span className="text-xs text-gray-400">{row.utilization_pct}%</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={"inline-flex rounded-full px-2 py-0.5 text-xs font-medium " + riskColors[row.risk_level]}>{row.risk_level}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="py-8 text-center">
            <Shield className="mx-auto h-10 w-10 text-gray-300" />
            <p className="mt-3 text-sm text-gray-400">No cross-analysis data available.</p>
            <p className="mt-1 text-xs text-gray-400">Backend endpoint /api/v1/identity/entitlement-review/cross-analysis may not be implemented yet.</p>
          </div>
        )}
      </div>

      {/* User-specific search */}
      <div className={cardCls}>
        <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">User Permission Review</h2>
        <div className="flex items-center gap-2">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
            <input
              aria-label="Search user ID or email"
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") searchUser(); }}
              placeholder="Search user ID or email..."
              className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
            />
          </div>
          <button onClick={searchUser} disabled={loading || !search} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1">
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />} Review
          </button>
        </div>

        {perms.length > 0 && (
          <>
            <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
              <div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Total Perms</span><p className="text-xl font-bold mt-1">{perms.length}</p></div>
              <div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Direct</span><p className="text-xl font-bold text-blue-600 mt-1">{perms.filter((p) => p.source === "direct").length}</p></div>
              <div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Unused 90d</span><p className="text-xl font-bold text-yellow-600 mt-1">{unused}</p></div>
              <div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Over-Privileged</span><p className="text-xl font-bold text-red-600 mt-1">{overPriv}</p></div>
            </div>

            <div className="mt-4 overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50">
                  <tr>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Resource</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Action</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Source</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Last Used</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Flags</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Recommendation</th>
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {perms.map((p) => {
                    const cfg = recConfig[p.recommendation];
                    const Icon = cfg.icon;
                    return (
                      <tr key={p.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                        <td className="px-4 py-3 font-mono text-xs">{p.resource}</td>
                        <td className="px-4 py-3 text-xs">{p.action}</td>
                        <td className="px-4 py-3"><span className={"text-xs " + (p.source === "direct" ? "text-blue-600" : "text-purple-600")}>{p.source}</span>{p.via_group && <span className="text-xs text-gray-400 ml-1">({p.via_group})</span>}</td>
                        <td className="px-4 py-3 text-xs text-gray-500">{p.last_used ? <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{p.last_used}</span> : "never"}</td>
                        <td className="px-4 py-3"><div className="flex gap-1">{p.unused_90d && <span className="px-1.5 py-0.5 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400">unused</span>}{p.over_privileged && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">over-priv</span>}</div></td>
                        <td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs " + cfg.color}><Icon className="w-3.5 h-3.5" /> {cfg.label}</span></td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
