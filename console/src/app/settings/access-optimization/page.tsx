"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Gauge, TrendingDown, Zap, Lightbulb, Shield, CheckCircle2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface OptimizationData {
  user_id: string;
  username: string;
  optimization_score: number;
  redundant_roles: { role: string; overlaps_with: string; overlap_pct: number }[];
  unused_paths: { path: string; last_accessed: string }[];
  suggestions: { action: string; impact: string; roles_affected: string[] }[];
}

export default function AccessOptimizationPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<OptimizationData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/access-optimization?user=${encodeURIComponent(user)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchData(search);
  }, [search, fetchData]);

  const scoreColor = data ? (data.optimization_score >= 80 ? "#10b981" : data.optimization_score >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Zap className="w-6 h-6 text-blue-500" /> {t("accessOptimization.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Identify redundant roles, unused access paths, and consolidation opportunities.</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" placeholder="Search by username or user ID..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <div className="space-y-4">
          {/* Optimization score gauge */}
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-6">
            <div className="relative w-24 h-24">
              <svg viewBox="0 0 64 64" className="w-full h-full">
                <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" />
                <circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${data.optimization_score * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.optimization_score}</span>
                <span className="text-[10px] text-gray-400">/100</span>
              </div>
            </div>
            <div>
              <h3 className="font-semibold">{data.username}</h3>
              <div className="flex items-center gap-4 mt-2 text-sm">
                <span className="flex items-center gap-1"><TrendingDown className="w-4 h-4 text-orange-400" /> {data.redundant_roles.length} redundant roles</span>
                <span className="flex items-center gap-1"><Shield className="w-4 h-4 text-gray-400" /> {data.unused_paths.length} unused paths</span>
                <span className="flex items-center gap-1"><Lightbulb className="w-4 h-4 text-yellow-400" /> {data.suggestions.length} suggestions</span>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Redundant roles */}
            <div className="rounded-lg border dark:border-gray-800">
              <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><TrendingDown className="w-4 h-4 text-orange-500" /> Redundant Roles ({data.redundant_roles.length})</h3></div>
              <div className="divide-y dark:divide-gray-800 max-h-64 overflow-y-auto">
                {data.redundant_roles.map((r, i) => (
                  <div key={i} className="px-4 py-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="font-mono text-xs">{r.role}</span>
                      <span className="px-2 py-0.5 rounded text-xs bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">{r.overlap_pct}% overlap</span>
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">Overlaps with: {r.overlaps_with}</p>
                  </div>
                ))}
                {data.redundant_roles.length === 0 && <p className="px-4 py-4 text-sm text-gray-500 flex items-center gap-1"><CheckCircle2 className="w-4 h-4 text-green-500" /> No redundant roles.</p>}
              </div>
            </div>

            {/* Unused paths */}
            <div className="rounded-lg border dark:border-gray-800">
              <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Shield className="w-4 h-4 text-gray-500" /> Unused Access Paths ({data.unused_paths.length})</h3></div>
              <div className="divide-y dark:divide-gray-800 max-h-64 overflow-y-auto">
                {data.unused_paths.map((p, i) => (
                  <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                    <span className="font-mono text-xs truncate">{p.path}</span>
                    <span className="text-xs text-gray-400 ml-2">{p.last_accessed}</span>
                  </div>
                ))}
                {data.unused_paths.length === 0 && <p className="px-4 py-4 text-sm text-gray-500 flex items-center gap-1"><CheckCircle2 className="w-4 h-4 text-green-500" /> No unused paths.</p>}
              </div>
            </div>
          </div>

          {/* Suggestions */}
          {data.suggestions.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4 bg-blue-50 dark:bg-blue-900/20">
              <h3 className="font-semibold mb-3 flex items-center gap-2"><Lightbulb className="w-4 h-4 text-blue-500" /> Consolidation Suggestions ({data.suggestions.length})</h3>
              <div className="space-y-2">
                {data.suggestions.map((s, i) => (
                  <div key={i} className="text-sm">
                    <div className="flex items-start gap-2">
                      <span className="text-blue-400 mt-0.5">•</span>
                      <div className="flex-1">
                        <span className="font-medium">{s.action}</span>
                        <span className="text-xs text-gray-500 ml-2">({s.impact})</span>
                        <div className="flex flex-wrap gap-1 mt-1">{s.roles_affected.map((r, ri) => <span key={ri} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{r}</span>)}</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No optimization data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to analyze access paths.</p>}
      {loading && <p className="text-sm text-gray-500">Loading...</p>}
    </div>
  );
}
