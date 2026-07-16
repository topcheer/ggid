"use client";

import { useState } from "react";
import { Copy, Trash2, Play, BarChart3 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface DedupStats {
  original_count: number;
  deduplicated_count: number;
  removed_count: number;
  duplicate_rate: number;
  top_duplicates: { fingerprint: string; count: number; sample: string }[];
}

const methods = [
  { value: "exact", label: "Exact Match" },
  { value: "fuzzy", label: "Fuzzy Match" },
  { value: "semantic", label: "Semantic Similarity" },
];

export default function EventDedupPage() {
  const t = useTranslations();

  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [method, setMethod] = useState("exact");
  const [stats, setStats] = useState<DedupStats | null>(null);
  const [loading, setLoading] = useState(false);

  const runDedup = async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/event-dedup", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ start_date: startDate, end_date: endDate, method }) });
      if (res.ok) setStats(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Copy className="w-6 h-6 text-purple-500" /> {t("eventDedup.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Identify and remove duplicate audit events within a date range.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Start Date</label><input aria-label="start Date" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-sm font-medium">End Date</label><input aria-label="end Date" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-sm font-medium">Method</label><select aria-label="method" value={method} onChange={(e) => setMethod(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">{methods.map((m) => <option key={m.value} value={m.value}>{m.label}</option>)}</select></div>
        </div>
        <button aria-label="Play" onClick={runDedup} disabled={loading || !startDate || !endDate} className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Processing..." : "Run Dedup"}</button>
      </div>

      {stats && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><BarChart3 className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Original</span><p className="text-xl font-bold">{stats.original_count}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Copy className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Deduplicated</span><p className="text-xl font-bold text-green-600">{stats.deduplicated_count}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Trash2 className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">Removed</span><p className="text-xl font-bold text-red-600">{stats.removed_count}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><span className="text-xl font-bold text-purple-600">{stats.duplicate_rate.toFixed(1)}%</span><div><span className="text-sm text-gray-500">Dup Rate</span></div></div>
          </div>

          {stats.top_duplicates.length > 0 && (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Fingerprint</th><th className="px-4 py-3 text-left font-medium">Occurrences</th><th className="px-4 py-3 text-left font-medium">Sample</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{stats.top_duplicates.map((d, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs text-purple-600">{d.fingerprint}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-bold">{d.count}</span></td><td className="px-4 py-3 text-xs text-gray-500 max-w-xs truncate">{d.sample}</td></tr>))}</tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
