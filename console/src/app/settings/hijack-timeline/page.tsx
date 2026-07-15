"use client";

import { useState, useCallback } from "react";
import { AlertTriangle, Search, ShieldAlert, CheckCircle, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface HijackEvent {
  id: string;
  timestamp: string;
  event_type: "suspicious_login" | "geo_jump" | "device_change" | "ip_change";
  description: string;
  details: string;
}

interface HijackData {
  user_id: string;
  username: string;
  confidence_score: number;
  events: HijackEvent[];
  recommended_actions: string[];
}

const eventIcons: Record<string, typeof AlertTriangle> = {
  suspicious_login: AlertTriangle,
  geo_jump: ShieldAlert,
  device_change: CheckCircle,
  ip_change: Clock,
};

export default function HijackTimelinePage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<HijackData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchTimeline = useCallback(async () => {
    if (!search) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/hijack-timeline?user_id=${encodeURIComponent(search)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [search]);

  const scoreColor = data ? (data.confidence_score >= 70 ? "#ef4444" : data.confidence_score >= 40 ? "#f59e0b" : "#10b981") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><AlertTriangle className="w-6 h-6 text-red-500" /> {t("hijackTimeline.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Account takeover detection with confidence scoring and recommended actions.</p>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input type="text" value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") fetchTimeline(); }} placeholder="user:alice or usr-xxxx" className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        <button onClick={fetchTimeline} disabled={loading || !search} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50">Analyze</button>
      </div>

      {data && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
            <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${(data.confidence_score / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-xl font-bold" style={{ color: scoreColor }}>{data.confidence_score.toFixed(0)}</span><span className="text-[9px] text-gray-400">confidence</span></div></div>
            <div><h3 className="font-semibold">{data.username}</h3><p className="text-sm text-gray-500 mt-1">Hijack confidence score</p>{data.confidence_score >= 70 && <span className="text-xs text-red-600 font-medium">High risk - immediate action recommended</span>}{data.confidence_score < 40 && <span className="text-xs text-green-600">Low risk</span>}</div>
          </div>

          <div className="relative pl-8">
            <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
            <div className="space-y-2">
              {data.events.map((evt) => { const Icon = eventIcons[evt.event_type] || AlertTriangle; return (
                <div key={evt.id} className="relative">
                  <div className="absolute -left-5 w-4 h-4 rounded-full bg-red-500 border-2 border-red-200" />
                  <div className="rounded-lg border dark:border-gray-800 p-3 ml-2">
                    <div className="flex items-center justify-between"><div className="flex items-center gap-2"><Icon className="w-4 h-4 text-red-500" /><span className="text-xs font-medium text-red-600">{evt.event_type.replace("_", " ")}</span></div><span className="text-xs text-gray-400">{evt.timestamp}</span></div>
                    <p className="text-sm mt-1">{evt.description}</p>
                    <p className="text-xs text-gray-400 mt-0.5">{evt.details}</p>
                  </div>
                </div>
              ); })}
              {data.events.length === 0 && <p className="text-sm text-gray-500 py-4 ml-2">No suspicious events detected.</p>}
            </div>
          </div>

          {data.recommended_actions.length > 0 && (
            <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><ShieldAlert className="w-4 h-4 text-orange-500" /> Recommended Actions</h3>
              <div className="space-y-1">{data.recommended_actions.map((a, i) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="text-xs text-gray-400">{i + 1}.</span><span>{a}</span></div>
              ))}</div>
            </div>
          )}
        </>
      )}
      {!data && !loading && search && <p className="text-sm text-gray-500 text-center py-8">Click Analyze to view timeline.</p>}
    </div>
  );
}
