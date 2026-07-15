"use client";

import { useState, useEffect, useCallback } from "react";
import { Shield, PieChart as PieIcon } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface BindingStats {
  bound: number;
  unbound: number;
  total: number;
  compliance_pct: number;
  binding_methods: { method: string; count: number }[];
  by_client: { client_id: string; client_name: string; bound: number; unbound: number; method: string }[];
}

const methodColors: Record<string, string> = { mTLS: "#3b82f6", DPoP: "#8b5cf6", PKI: "#10b981", none: "#ef4444" };

export default function TokenBindingStatsPage() {
  const t = useTranslations();

  const [data, setData] = useState<BindingStats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/token-binding-stats", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const gaugeColor = data ? (data.compliance_pct >= 80 ? "#10b981" : data.compliance_pct >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  const totalMethods = data?.binding_methods.reduce((s, m) => s + m.count, 0) || 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("tokenBindingStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Token sender-constraint binding compliance across all OAuth clients.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><PieIcon className="w-4 h-4" /> Bound vs Unbound</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90"><circle cx={32} cy={32} r={28} fill="none" stroke="#ef4444" strokeWidth={8} strokeDasharray={`${(data.unbound / data.total) * 176} 176`} /><circle cx={32} cy={32} r={28} fill="none" stroke="#10b981" strokeWidth={8} strokeDasharray={`${(data.bound / data.total) * 176} 176`} strokeDashoffset={`${-(data.bound / data.total) * 176}`} /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold">{data.total}</span><span className="text-[9px] text-gray-400">tokens</span></div></div>
                <div className="space-y-1"><div className="flex items-center gap-2 text-xs"><span className="w-3 h-3 rounded bg-green-500" /><span>Bound: <strong>{data.bound}</strong></span></div><div className="flex items-center gap-2 text-xs"><span className="w-3 h-3 rounded bg-red-500" /><span>Unbound: <strong>{data.unbound}</strong></span></div></div>
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
              <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(data.compliance_pct / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold" style={{ color: gaugeColor }}>{data.compliance_pct.toFixed(0)}%</span><span className="text-[9px] text-gray-400">compliance</span></div></div>
              <div><span className="text-sm text-gray-500">Compliance Rate</span><p className="text-xs text-gray-400 mt-1">sender-constrained tokens</p></div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Binding Methods</h3>
              <div className="space-y-2">{data.binding_methods.map((m) => (
                <div key={m.method} className="flex items-center gap-2"><span className="w-3 h-3 rounded" style={{ background: methodColors[m.method] || "#ccc" }} /><span className="text-sm flex-1 font-mono">{m.method}</span><span className="font-bold text-sm">{m.count}</span><span className="text-xs text-gray-400">({((m.count / totalMethods) * 100).toFixed(0)}%)</span></div>
              ))}</div>
            </div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">Bound</th><th className="px-4 py-3 text-left font-medium">Unbound</th><th className="px-4 py-3 text-left font-medium">Method</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.by_client.map((c) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.client_name}</span><p className="text-xs text-gray-400 font-mono">{c.client_id}</p></td><td className="px-4 py-3 font-bold text-green-600">{c.bound}</td><td className="px-4 py-3 font-bold text-red-600">{c.unbound}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs font-mono" style={{ background: (methodColors[c.method] || "#ccc") + "20", color: methodColors[c.method] || "#ccc" }}>{c.method}</span></td></tr>))}</tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
