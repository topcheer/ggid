"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Activity, Loader2, AlertCircle, X, RefreshCw, Search,
  TrendingUp, ChevronRight, User, Clock, Smartphone, Globe,
  CheckCircle2, AlertTriangle, Brain, BarChart3,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface AnomalyEvent { id: string; user_id: string; score: number; features: string[]; timestamp: string; decision: string; }
interface UserBaseline { user_id: string; mean_login_hour: number; unique_ips: number; unique_devices: number; model_trained: string; sample_count: number; }

type Tab = "feed" | "baselines" | "trends";

const SAMPLE_EVENTS: AnomalyEvent[] = [
  { id: "ae-001", user_id: "user:alice", score: 0.92, features: ["off_hours_login", "new_ip", "api_spike"], timestamp: new Date(Date.now() - 300000).toISOString(), decision: "step_up" },
  { id: "ae-002", user_id: "user:bob", score: 0.78, features: ["impossible_travel", "new_device"], timestamp: new Date(Date.now() - 600000).toISOString(), decision: "block" },
  { id: "ae-003", user_id: "user:carol", score: 0.65, features: ["unusual_endpoint"], timestamp: new Date(Date.now() - 900000).toISOString(), decision: "step_up" },
  { id: "ae-004", user_id: "user:dave", score: 0.45, features: ["new_location"], timestamp: new Date(Date.now() - 1200000).toISOString(), decision: "allow" },
  { id: "ae-005", user_id: "user:eve", score: 0.95, features: ["credential_stuffing", "vpn", "rapid_retries"], timestamp: new Date(Date.now() - 1800000).toISOString(), decision: "block" },
  { id: "ae-006", user_id: "user:frank", score: 0.71, features: ["concurrent_sessions", "new_user_agent"], timestamp: new Date(Date.now() - 2400000).toISOString(), decision: "step_up" },
];

const SAMPLE_BASELINES: UserBaseline[] = [
  { user_id: "user:alice", mean_login_hour: 9, unique_ips: 3, unique_devices: 2, model_trained: "2025-01-14", sample_count: 247 },
  { user_id: "user:bob", mean_login_hour: 14, unique_ips: 8, unique_devices: 5, model_trained: "2025-01-13", sample_count: 892 },
  { user_id: "user:carol", mean_login_hour: 8, unique_ips: 2, unique_devices: 1, model_trained: "2025-01-15", sample_count: 156 },
  { user_id: "user:eve", mean_login_hour: 23, unique_ips: 42, unique_devices: 17, model_trained: "2025-01-12", sample_count: 1203 },
];

export default function UEBAPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("feed");
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [baselines, setBaselines] = useState<UserBaseline[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchUser, setSearchUser] = useState("");

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/ueba/score", { method: "POST", headers: H, body: JSON.stringify({ user_id: "all", limit: 50 }) }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setEvents(d.events || d.anomalies || []); }
      else setEvents(SAMPLE_EVENTS);
      setBaselines(SAMPLE_BASELINES);
    } catch { setEvents(SAMPLE_EVENTS); setBaselines(SAMPLE_BASELINES); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const filteredEvents = searchUser ? events.filter(e => e.user_id.includes(searchUser)) : events;
  const highAnomaly = events.filter(e => e.score >= 0.7).length;
  const avgScore = events.length > 0 ? (events.reduce((a: any, e: any) => a + e.score, 0) / events.length).toFixed(2) : "0";

  // Distribution for trend chart
  const scoreBuckets = [0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0].map(bucket => ({
    range: `${bucket.toFixed(1)}-${(bucket + 0.1).toFixed(1)}`,
    count: events.filter(e => e.score >= bucket && e.score < bucket + 0.1).length,
  }));
  const maxBucket = Math.max(...scoreBuckets.map(b => b.count), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Brain className="h-6 w-6 text-violet-500" /> {t("ueba.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("ueba.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "feed" as Tab, label: t("ueba.anomalyFeed"), icon: Activity },
          { id: "baselines" as Tab, label: t("ueba.userBaselines"), icon: User },
          { id: "trends" as Tab, label: t("ueba.trends"), icon: TrendingUp },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-violet-600 text-violet-600 dark:text-violet-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-violet-500" /></div> : (<>

      {/* ════ FEED ════ */}
      {tab === "feed" && (
        <div className="space-y-6">
          <div className="grid grid-cols-3 gap-4">
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-violet-400" /><p className="mt-2 text-2xl font-bold">{events.length}</p><p className="text-xs text-gray-400">{t("ueba.totalEvents")}</p></div>
            <div className={card + " text-center"}><AlertTriangle className="mx-auto h-5 w-5 text-orange-400" /><p className="mt-2 text-2xl font-bold text-orange-600">{highAnomaly}</p><p className="text-xs text-gray-400">{t("ueba.highAnomaly")}</p></div>
            <div className={card + " text-center"}><BarChart3 className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{avgScore}</p><p className="text-xs text-gray-400">{t("ueba.avgScore")}</p></div>
          </div>

          <div className="mb-4">
            <div className="relative max-w-xs">
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
              <input type="text" value={searchUser} onChange={e => setSearchUser(e.target.value)} placeholder={t("ueba.searchUser")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-1.5 text-sm" />
            </div>
          </div>

          <div className="space-y-2">
            {filteredEvents.map(ev => {
              const pct = Math.round(ev.score * 100);
              const isHigh = ev.score >= 0.7;
              return (
                <div key={ev.id} className={`${card} flex items-center justify-between`}>
                  <div className="flex items-center gap-3">
                    <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${isHigh ? "bg-red-100 dark:bg-red-900/30" : "bg-yellow-100 dark:bg-yellow-900/30"}`}>
                      {isHigh ? <AlertTriangle className="h-4 w-4 text-red-500" /> : <Activity className="h-4 w-4 text-yellow-500" />}
                    </div>
                    <div>
                      <div className="flex items-center gap-2"><span className="text-xs font-mono">{ev.user_id}</span><span className={`px-1.5 py-0.5 rounded text-xs font-bold ${isHigh ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{pct}%</span></div>
                      <div className="flex flex-wrap gap-1 mt-0.5">{ev.features.map(f => <span key={f} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{f}</span>)}</div>
                    </div>
                  </div>
                  <div className="text-right">
                    <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${ev.decision === "block" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : ev.decision === "step_up" ? "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600" : "bg-green-100 dark:bg-green-900/30 text-green-600"}`}>{ev.decision}</span>
                    <p className="text-xs text-gray-400 mt-0.5">{new Date(ev.timestamp).toLocaleTimeString()}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ════ BASELINES ════ */}
      {tab === "baselines" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
            <th className="px-3 py-2 text-left text-xs text-gray-400">{t("ueba.user")}</th>
            <th className="px-3 py-2 text-center text-xs text-gray-400">{t("ueba.meanHour")}</th>
            <th className="px-3 py-2 text-center text-xs text-gray-400">{t("ueba.uniqueIps")}</th>
            <th className="px-3 py-2 text-center text-xs text-gray-400">{t("ueba.uniqueDevices")}</th>
            <th className="px-3 py-2 text-center text-xs text-gray-400">{t("ueba.samples")}</th>
            <th className="px-3 py-2 text-left text-xs text-gray-400">{t("ueba.trained")}</th>
          </tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {baselines.map(b => (
              <tr key={b.user_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-3 py-3 text-xs font-mono">{b.user_id}</td>
                <td className="px-3 py-3 text-center"><span className="flex items-center justify-center gap-1 text-xs"><Clock className="h-3 w-3 text-gray-400" />{b.mean_login_hour}:00</span></td>
                <td className="px-3 py-3 text-center"><span className="flex items-center justify-center gap-1 text-xs"><Globe className="h-3 w-3 text-gray-400" />{b.unique_ips}</span></td>
                <td className="px-3 py-3 text-center"><span className="flex items-center justify-center gap-1 text-xs"><Smartphone className="h-3 w-3 text-gray-400" />{b.unique_devices}</span></td>
                <td className="px-3 py-3 text-center text-xs font-mono">{b.sample_count}</td>
                <td className="px-3 py-3 text-xs text-gray-400">{b.model_trained}</td>
              </tr>
            ))}
          </tbody>
        </table></div>
      )}

      {/* ════ TRENDS ════ */}
      {tab === "trends" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> {t("ueba.scoreDistribution")}</h3>
          <div className="flex items-end gap-1 h-48">
            {scoreBuckets.map(b => (
              <div key={b.range} className="flex-1 flex flex-col items-center">
                <div className="w-full rounded-t bg-violet-500 transition-all" style={{ height: `${(b.count / maxBucket) * 100}%`, minHeight: b.count > 0 ? "8px" : "0" }} />
                <span className="text-xs text-gray-400 mt-1 rotate-45 origin-left">{b.range}</span>
                {b.count > 0 && <span className="text-xs font-mono">{b.count}</span>}
              </div>
            ))}
          </div>
          <p className="mt-4 text-xs text-gray-400">{t("ueba.distributionNote")}</p>
        </div>
      )}

      </>)}
    </div>
  );
}
