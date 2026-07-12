"use client";
import { useState, useEffect, useCallback } from "react";
import { Activity, Clock, Globe, Users } from "lucide-react";

interface SessionData { active_count: number; avg_duration_minutes: number; revocation_rate: number; peak_concurrent: number; peak_hour: string; by_platform: { platform: string; count: number }[]; by_location: { location: string; count: number }[]; top_users: { user_id: string; username: string; session_count: number; avg_duration: number }[]; }

export default function SessionAnalyticsPage() {
  const [data, setData] = useState<SessionData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/session-analytics", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const platformColors = ["#3b82f6", "#10b981", "#f59e0b", "#8b5cf6", "#ef4444", "#06b6d4"];
  const totalPlatform = data ? data.by_platform.reduce((s, p) => s + p.count, 0) || 1 : 1;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> Session Analytics</h1><p className="text-sm text-gray-500 mt-1">Monitor active sessions, duration, platform distribution, and peak usage.</p></div>

      {data && (<>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Activity className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Active</span><p className="text-xl font-bold">{data.active_count}</p></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Clock className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Avg Duration</span><p className="text-xl font-bold">{data.avg_duration_minutes}m</p></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Users className="w-8 h-8 text-purple-500" /><div><span className="text-sm text-gray-500">Peak Concurrent</span><p className="text-xl font-bold">{data.peak_concurrent}</p></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><span className="text-xl font-bold text-orange-600">{data.revocation_rate.toFixed(1)}%</span><div><span className="text-sm text-gray-500">Revocation Rate</span></div></div>
        </div>

        {data.peak_hour && <div className="rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 p-3 text-sm text-blue-700 dark:text-blue-400">Peak concurrent sessions at {data.peak_hour}</div>}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">By Platform</h3><div className="flex items-center gap-4"><div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{data.by_platform.map((p, i) => { let off = 0; for (let j = 0; j < i; j++) off += data.by_platform[j].count / totalPlatform; const pct = p.count / totalPlatform; return <circle key={i} cx={32} cy={32} r={28} fill="none" stroke={platformColors[i % 6]} strokeWidth={8} strokeDasharray={pct * 176 + " 176"} strokeDashoffset={-off * 176} />; })}</svg></div><div className="space-y-1">{data.by_platform.map((p, i) => (<div key={p.platform} className="flex items-center gap-2 text-xs"><span className="w-3 h-3 rounded-full" style={{ background: platformColors[i % 6] }} /><span>{p.platform}</span><span className="font-bold ml-auto">{p.count}</span></div>))}</div></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 flex items-center gap-1"><Globe className="w-4 h-4 text-gray-400" /> By Location</h3><div className="space-y-1">{data.by_location.slice(0, 6).map((l) => (<div key={l.location} className="flex items-center gap-2 text-xs"><span className="w-20 truncate">{l.location}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-4"><div className="h-full rounded-full bg-blue-500" style={{ width: (l.count / (data.by_location[0]?.count || 1)) * 100 + "%" }} /></div><span className="font-bold w-8 text-right">{l.count}</span></div>))}</div></div>
        </div>

        <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Sessions</th><th className="px-4 py-3 text-left font-medium">Avg Duration</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.top_users.map((u) => (<tr key={u.user_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{u.username}</span><p className="text-xs text-gray-400 font-mono">{u.user_id}</p></td><td className="px-4 py-3 font-bold">{u.session_count}</td><td className="px-4 py-3 text-xs text-gray-500">{u.avg_duration}m</td></tr>))}</tbody></table></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
