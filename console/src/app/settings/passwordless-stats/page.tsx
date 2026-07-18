"use client";

import { useState, useEffect, useCallback } from "react";
import { Fingerprint, Clock, TrendingDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Stats {
  method_distribution: { method: string; count: number }[];
  success_rate: number;
  avg_completion_time_ms: number;
  abandonment_rate: number;
  by_device_type: { device: string; attempts: number; success_pct: number }[];
}

const methodColors: Record<string, string> = {
  magic_link: "#3b82f6", passkey: "#8b5cf6", biometric: "#10b981",
};

export default function PasswordlessStatsPage() {
  const t = useTranslations();

  const [data, setData] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/passwordless-stats", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const total = data?.method_distribution.reduce((s: any, m: any) => s + m.count, 0) || 1;
  const gaugeColor = data ? (data.success_rate >= 80 ? "#10b981" : data.success_rate >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Fingerprint className="w-6 h-6 text-purple-500" /> {t("passwordlessStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Passwordless authentication adoption and success metrics.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Method Distribution</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{(() => { let offset = 0; return data.method_distribution.map((m: any) => { const pct = m.count / total; const dash = pct * 176; const circle = <circle key={m.method} cx={32} cy={32} r={28} fill="none" stroke={methodColors[m.method] || "#ccc"} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-offset * 176} />; offset += pct; return circle; }); })()}</svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold">{total}</span><span className="text-[9px] text-gray-400">total</span></div></div>
                <div className="space-y-1">{data.method_distribution.map((m: any) => (<div key={m.method} className="flex items-center gap-2 text-xs"><span className="w-3 h-3 rounded" style={{ background: methodColors[m.method] || "#ccc" }} /><span className="capitalize">{m.method.replace("_", " ")}</span><span className="font-bold ml-auto">{m.count}</span></div>))}</div>
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
              <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(data.success_rate / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold" style={{ color: gaugeColor }}>{data.success_rate.toFixed(0)}%</span><span className="text-[9px] text-gray-400">success</span></div></div>
              <div><span className="text-sm text-gray-500">Success Rate</span><p className="text-xs text-gray-400 mt-1">all passwordless methods</p></div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
              <div className="flex items-center gap-2"><Clock className="w-5 h-5 text-blue-500" /><div><span className="text-sm text-gray-500">Avg Completion</span><p className="text-lg font-bold">{data.avg_completion_time_ms}ms</p></div></div>
              <div className="flex items-center gap-2"><TrendingDown className="w-5 h-5 text-red-500" /><div><span className="text-sm text-gray-500">Abandonment</span><div className="flex items-center gap-2 mt-1"><div className="w-24 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full bg-red-500 rounded-full" style={{ width: `${data.abandonment_rate}%` }} /></div><span className="text-sm font-bold text-red-600">{data.abandonment_rate.toFixed(1)}%</span></div></div></div>
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">By Device Type</h3>
            <div className="space-y-2">{data.by_device_type.map((d: any) => (
              <div key={d.device} className="flex items-center gap-3"><span className="text-sm w-20 capitalize">{d.device}</span><span className="text-xs text-gray-500 w-16">{d.attempts} tries</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5 overflow-hidden"><div className="h-full bg-purple-500 rounded-full" style={{ width: `${d.success_pct}%` }} /></div><span className="text-xs font-bold w-10 text-right">{d.success_pct}%</span></div>
            ))}</div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
