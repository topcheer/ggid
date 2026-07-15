"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { ShieldAlert, KeyRound, Globe, Monitor, AlertTriangle, Filter } from "lucide-react";

interface TokenReuse {
  id: string;
  token_masked: string;
  user_id: string;
  username: string;
  ip_address: string;
  country: string;
  user_agent: string;
  first_seen: string;
  last_seen: string;
  reuse_count: number;
  risk_level: "low" | "medium" | "high" | "critical";
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function TokenReusePage() {
  const t = useTranslations();
  const [reuses, setReuses] = useState<TokenReuse[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterRisk, setFilterRisk] = useState("all");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/token-reuse", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setReuses(data.reuses || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = filterRisk === "all" ? reuses : reuses.filter((r) => r.risk_level === filterRisk);
  const critical = reuses.filter((r) => r.risk_level === "critical").length;
  const high = reuses.filter((r) => r.risk_level === "high").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-red-500" />{t("tokenReuse.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Detect suspicious token reuse across different IPs and user agents.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Detections</span><p className="text-2xl font-bold mt-1">{reuses.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Critical</span><p className="text-2xl font-bold mt-1 text-red-600">{critical}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">High</span><p className="text-2xl font-bold mt-1 text-orange-600">{high}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Unique Users</span><p className="text-2xl font-bold mt-1">{new Set(reuses.map((r) => r.user_id)).size}</p></div>
      </div>

      {critical > 0 && (
        <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2">
          <AlertTriangle className="w-5 h-5 text-red-500" />
          <span className="font-semibold text-red-700 dark:text-red-400">{critical} critical token reuse incident{critical > 1 ? "s" : ""} require immediate investigation</span>
        </div>
      )}

      {/* Filter */}
      <div className="flex items-center gap-3">
        <Filter className="w-4 h-4 text-gray-400" />
        <select value={filterRisk} onChange={(e) => setFilterRisk(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="all">All Risk Levels</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
        <span className="text-sm text-gray-500">{filtered.length} incidents</span>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium">Token</th>
              <th className="px-4 py-3 text-left font-medium">User</th>
              <th className="px-4 py-3 text-left font-medium">IP Address</th>
              <th className="px-4 py-3 text-left font-medium">Country</th>
              <th className="px-4 py-3 text-left font-medium">User Agent</th>
              <th className="px-4 py-3 text-left font-medium">Reuse Count</th>
              <th className="px-4 py-3 text-left font-medium">Last Seen</th>
              <th className="px-4 py-3 text-left font-medium">Risk</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((r) => (
              <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs text-gray-500">{r.token_masked}</td>
                <td className="px-4 py-3"><span className="font-medium">{r.username}</span></td>
                <td className="px-4 py-3"><span className="flex items-center gap-1 font-mono text-xs"><Globe className="w-3 h-3 text-gray-400" />{r.ip_address}</span></td>
                <td className="px-4 py-3 text-xs">{r.country}</td>
                <td className="px-4 py-3 max-w-xs truncate" title={r.user_agent}><span className="flex items-center gap-1 text-xs text-gray-500"><Monitor className="w-3 h-3 flex-shrink-0" />{r.user_agent}</span></td>
                <td className="px-4 py-3"><span className="font-bold text-orange-600">{r.reuse_count}</span></td>
                <td className="px-4 py-3 text-gray-500">{r.last_seen}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[r.risk_level]}`}>{r.risk_level}</span></td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No token reuse incidents found.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
