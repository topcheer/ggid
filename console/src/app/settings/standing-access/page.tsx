"use client";

import { useState, useEffect, useCallback } from "react";
import { Clock, Zap, TrendingDown, CheckCircle2, ArrowRight } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface StandingAccess {
  id: string;
  user_id: string;
  username: string;
  resource: string;
  access_type: string;
  granted_at: string;
  last_used: string;
  days_since_use: number;
  jit_recommended: boolean;
  jit_role: string;
}

export default function StandingAccessPage() {
  const t = useTranslations();

  const [entries, setEntries] = useState<StandingAccess[]>([]);
  const [loading, setLoading] = useState(false);
  const [applyingId, setApplyingId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/standing-access", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setEntries(data.entries || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const applyJIT = async (id: string) => {
    setApplyingId(id);
    try {
      await fetch(`/api/v1/policy/standing-access/${id}/convert-jit`, {
        method: "POST",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setEntries((prev) => prev.filter((e) => e.id !== id));
    } catch { /* noop */ }
    finally { setApplyingId(null); }
  };

  const jitRecommended = entries.filter((e) => e.jit_recommended);
  const totalEntries = entries.length;
  const unused90 = entries.filter((e) => e.days_since_use >= 90).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Clock className="w-6 h-6 text-blue-500" /> {t("standingAccess.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Identify standing access and convert to just-in-time provisioning.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Total Standing</span><Clock className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{totalEntries}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">JIT Recommended</span><Zap className="w-5 h-5 text-yellow-400" /></div>
          <p className="text-2xl font-bold mt-1 text-yellow-600">{jitRecommended.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Unused 90+ Days</span><TrendingDown className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1 text-orange-600">{unused90}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Conversion Rate</span><CheckCircle2 className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1 text-green-600">{totalEntries > 0 ? Math.round((jitRecommended.length / totalEntries) * 100) : 0}%</p>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium">User</th>
              <th className="px-4 py-3 text-left font-medium">Resource</th>
              <th className="px-4 py-3 text-left font-medium">Access Type</th>
              <th className="px-4 py-3 text-left font-medium">Granted</th>
              <th className="px-4 py-3 text-left font-medium">Last Used</th>
              <th className="px-4 py-3 text-left font-medium">Days Since Use</th>
              <th className="px-4 py-3 text-left font-medium">JIT Recommendation</th>
              <th className="px-4 py-3 text-left font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {entries.map((e) => (
              <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium">{e.username}</td>
                <td className="px-4 py-3 font-mono text-xs">{e.resource}</td>
                <td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{e.access_type}</span></td>
                <td className="px-4 py-3 text-gray-500">{e.granted_at}</td>
                <td className="px-4 py-3 text-gray-500">{e.last_used || "Never"}</td>
                <td className="px-4 py-3">
                  <span className={`font-bold ${e.days_since_use >= 180 ? "text-red-600" : e.days_since_use >= 90 ? "text-orange-600" : "text-yellow-600"}`}>{e.days_since_use}</span>
                </td>
                <td className="px-4 py-3">
                  {e.jit_recommended ? (
                    <span className="flex items-center gap-1 text-xs text-yellow-600"><Zap className="w-3 h-3" /> Convert to {e.jit_role}</span>
                  ) : (
                    <span className="text-xs text-gray-400">-</span>
                  )}
                </td>
                <td className="px-4 py-3">
                  {e.jit_recommended && (
                    <button onClick={() => applyJIT(e.id)} disabled={applyingId === e.id} className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-50 flex items-center gap-1">
                      <ArrowRight className="w-3 h-3" />
                      {applyingId === e.id ? "Converting..." : "Apply JIT"}
                    </button>
                  )}
                </td>
              </tr>
            ))}
            {entries.length === 0 && !loading && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No standing access entries found.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
