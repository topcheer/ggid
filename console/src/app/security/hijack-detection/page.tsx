"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldAlert, Loader2, AlertCircle, X, XCircle, MapPin, Activity, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface GeoLocation { ip: string; city: string; country: string; timestamp: string; }
interface SuspiciousSession {
  session_id: string; user_id: string; username: string;
  concurrent_ips: string[]; geo_velocity_kmh: number;
  locations: GeoLocation[]; risk_score: number;
  detected_at: string; reason: string;
}

function riskColor(score: number): string {
  const t = useTranslations();

  if (score >= 80) return "text-red-600";
  if (score >= 60) return "text-orange-600";
  return "text-yellow-600";
}

export default function HijackDetectionPage() {
  const t = useTranslations();  const { apiFetch } = useApi();
  const [sessions, setSessions] = useState<SuspiciousSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [terminating, setTerminating] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setSessions(await apiFetch<SuspiciousSession[]>("/api/v1/auth/sessions/hijack-detection").catch(() => [])); }
      catch { setError("Failed to load hijack detection data"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleTerminate = async (sessionId: string) => {
    setTerminating(sessionId);
    try { await apiFetch(`/api/v1/auth/sessions/${sessionId}/terminate`, { method: "POST" }); setSessions((p) => p.filter((s: any) => s.session_id !== sessionId)); }
    catch { setError("Terminate failed"); }
    finally { setTerminating(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldAlert className="h-6 w-6 text-red-600" /> {t("securityHijackDetection.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Suspicious sessions detected via concurrent IP usage and geo-velocity analysis.</p>
      </div>

      {sessions.length > 0 && <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-900/20"><ShieldAlert className="h-5 w-5 text-red-600 shrink-0" /><span className="text-sm text-red-700 dark:text-red-400">{sessions.length} suspicious session{sessions.length > 1 ? "s" : ""} detected.</span></div>}

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-600" /></div>
      : sessions.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><ShieldAlert className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">No suspicious sessions detected.</p></div></div>
      ) : (
        <div className="space-y-3">{sessions.map((s: any) => (
          <div key={s.session_id} className={`${cardCls} border-l-4 border-l-red-400`}>
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{s.username}</span><span className={`flex items-center gap-0.5 text-lg font-bold ${riskColor(s.risk_score)}`}><Activity className="h-3 w-3" />{s.risk_score}</span><span className="rounded bg-red-100 px-2 py-0.5 text-xs font-medium text-red-600 dark:bg-red-900/30">{s.reason}</span></div>
                <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-gray-400">
                  <span className="flex items-center gap-1"><Zap className="h-3 w-3" />Geo velocity: <span className={`font-medium ${s.geo_velocity_kmh > 1000 ? "text-red-500" : "text-orange-500"}`}>{s.geo_velocity_kmh.toFixed(0)} km/h</span></span>
                  <span>Concurrent IPs: {s.concurrent_ips.join(", ")}</span>
                </div>
                {/* Geo trail */}
                {s.locations.length > 1 && (
                  <div className="mt-3 flex items-center gap-2">{s.locations.map((loc: any, i: number) => (<React.Fragment key={i}><div className="flex items-center gap-1 rounded bg-gray-50 px-2 py-1 text-xs dark:bg-gray-900"><MapPin className="h-3 w-3 text-gray-400" /><span className="text-gray-600 dark:text-gray-300">{loc.city}, {loc.country}</span><span className="text-gray-400">{new Date(loc.timestamp).toLocaleTimeString()}</span></div>{i < s.locations.length - 1 && <span className="text-gray-300">→</span>}</React.Fragment>))}</div>
                )}
              </div>
              <button onClick={() => handleTerminate(s.session_id)} disabled={terminating === s.session_id} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{terminating === s.session_id ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />} Terminate</button>
            </div>
          </div>
        ))}</div>
      )}
    </div>
  );
}
