"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { AlertTriangle, Activity, Check, X, Filter } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface AnomalyEvent {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  user: string;
  timestamp: string;
  confidence: number;
  detail: string;
  status: "active" | "acknowledged" | "dismissed";
}

const sevColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function AnomalyDetectionPage() {
  const t = useTranslations();
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filterType, setFilterType] = useState("");
  const [filterSeverity, setFilterSeverity] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/v1/audit/anomaly-detection", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json(); setEvents(d.events || d || []);
    } catch (err) { setError(err instanceof Error ? err.message : t("anomalyDetect.error")); }
    finally { setLoading(false); }
  }, [t]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const updateStatus = async (id: string, status: "acknowledged" | "dismissed") => {
    setEvents(events.map((e) => e.id === id ? { ...e, status } : e));
    try { await fetch("/api/v1/audit/anomaly-detection/" + id, { method: "PATCH", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ status }) }); } catch { /* noop */ }
  };

  const filtered = events.filter((e) => {
    if (filterType && e.type !== filterType) return false;
    if (filterSeverity && e.severity !== filterSeverity) return false;
    return true;
  });
  const activeCount = events.filter((e) => e.status === "active").length;
  const types = [...new Set(events.map((e) => e.type))];

  if (error) return (
    <div className="p-8">
      <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400 text-sm font-medium">{t("common.error")}: {error}</p>
        <button aria-label="action" onClick={fetchData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("common.retry")}</button>
      </div>
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><AlertTriangle className="w-6 h-6 text-red-500" /> {t("anomalyDetect.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("anomalyDetect.subtitle")}</p></div>
        <span className="flex items-center gap-2 text-sm"><span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" /><span className="text-gray-500">{t("anomalyDetect.live")}</span><span className="font-bold text-red-600">{t("anomalyDetect.activeCount").replace("{count}", String(activeCount))}</span></span>
      </div>

      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-gray-400" />
        <select aria-label="Filter" value={filterType} onChange={(e) => setFilterType(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">{t("anomalyDetect.allTypes")}</option>{types.map((type) => <option key={type} value={type}>{type}</option>)}</select>
        <select aria-label="Filter" value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">{t("anomalyDetect.allSeverities")}</option><option value="low">{t("anomalyDetect.low")}</option><option value="medium">{t("anomalyDetect.medium")}</option><option value="high">{t("anomalyDetect.high")}</option><option value="critical">{t("anomalyDetect.critical")}</option></select>
        <span className="text-sm text-gray-500">{t("anomalyDetect.eventsCount").replace("{count}", String(filtered.length))}</span>
      </div>

      <div className="space-y-2">
        {filtered.map((e) => (
          <div key={e.id} className="rounded-lg border dark:border-gray-800 p-3 flex items-center gap-4">
            <div className={"w-1 self-stretch rounded " + (e.severity === "critical" ? "bg-red-500" : e.severity === "high" ? "bg-orange-500" : e.severity === "medium" ? "bg-yellow-500" : "bg-gray-400")} />
            <div className="flex-1"><div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs " + sevColors[e.severity]}>{e.severity}</span><span className="text-xs font-mono text-gray-500">{e.type}</span>{e.status !== "active" && <span className="text-xs text-gray-400 italic">({e.status})</span>}</div><p className="text-sm mt-1">{e.detail}</p><div className="flex items-center gap-3 text-xs text-gray-400 mt-1"><span>{t("anomalyDetect.user").replace("{user}", e.user)}</span><span>{t("anomalyDetect.confidence").replace("{value}", String(e.confidence))}</span><span>{e.timestamp}</span></div></div>
            {e.status === "active" && <div className="flex gap-1"><button onClick={() => updateStatus(e.id, "acknowledged")} className="p-1.5 rounded hover:bg-green-50 dark:hover:bg-green-900/20 text-green-600" title={t("anomalyDetect.acknowledge")}><Check className="w-4 h-4" /></button><button onClick={() => updateStatus(e.id, "dismissed")} className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-400" title={t("anomalyDetect.dismiss")}><X className="w-4 h-4" /></button></div>}
          </div>
        ))}
        {filtered.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("anomalyDetect.noAnomalies")}</p>}
      </div>
    </div>
  );
}
