"use client";

import { useState, useEffect, useCallback } from "react";
import { Smartphone, Gauge, Clock } from "lucide-react";

interface MFAStats {
  method_distribution: { method: string; count: number }[];
  enrollment_rate: number;
  avg_methods_per_user: number;
  pending_enrollments: { user_id: string; username: string; method: string; initiated_at: string }[];
}

const methodColors: Record<string, string> = {
  totp: "#3b82f6", sms: "#8b5cf6", email: "#10b981", webauthn: "#f59e0b", backup: "#ef4444",
};

export default function MFAEnrollmentStatsPage() {
  const [data, setData] = useState<MFAStats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/mfa-enrollment-stats", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const total = data?.method_distribution.reduce((s, d) => s + d.count, 0) || 1;
  const gaugeColor = data ? (data.enrollment_rate >= 80 ? "#10b981" : data.enrollment_rate >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Smartphone className="w-6 h-6 text-blue-500" /> MFA Enrollment Stats</h1>
        <p className="text-sm text-gray-500 mt-1">Multi-factor authentication enrollment across the organization.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
              <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(data.enrollment_rate / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-xl font-bold" style={{ color: gaugeColor }}>{data.enrollment_rate.toFixed(0)}%</span><span className="text-[9px] text-gray-400">enrolled</span></div></div>
              <div><span className="text-sm text-gray-500">Enrollment Rate</span><p className="text-xs text-gray-400 mt-1">of all active users</p></div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><Gauge className="w-8 h-8 text-purple-500" /><div><span className="text-sm text-gray-500">Avg Methods/User</span><p className="text-xl font-bold mt-1">{data.avg_methods_per_user.toFixed(2)}</p></div></div>

            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><Clock className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">Pending</span><p className="text-xl font-bold text-orange-600 mt-1">{data.pending_enrollments.length}</p></div></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">Method Distribution</h3>
            <div className="space-y-2">{data.method_distribution.map((d) => (
              <div key={d.method} className="flex items-center gap-2"><span className="text-xs font-mono w-24">{d.method}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-6 overflow-hidden"><div className="h-full rounded-full" style={{ width: `${(d.count / total) * 100}%`, background: methodColors[d.method] || "#ccc" }} /></div><span className="text-sm font-bold w-12 text-right">{d.count}</span><span className="text-xs text-gray-400 w-10">{((d.count / total) * 100).toFixed(0)}%</span></div>
            ))}</div>
          </div>

          {data.pending_enrollments.length > 0 && (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Method</th><th className="px-4 py-3 text-left font-medium">Initiated</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{data.pending_enrollments.map((p) => (<tr key={p.user_id + p.method} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{p.username}</span><p className="text-xs text-gray-400 font-mono">{p.user_id}</p></td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs" style={{ background: (methodColors[p.method] || "#ccc") + "20", color: methodColors[p.method] || "#ccc" }}>{p.method}</span></td><td className="px-4 py-3 text-xs text-gray-500">{p.initiated_at}</td></tr>))}</tbody>
              </table>
            </div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
