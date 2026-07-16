"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { Activity, Server, Zap, AlertTriangle, CheckCircle, XCircle } from "lucide-react";

interface Destination {
  id: string;
  name: string;
  endpoint: string;
  status: "healthy" | "degraded" | "down";
  connectivity: boolean;
  latency_ms: number;
  throughput_per_sec: number;
  error_rate: number;
  last_success: string;
}

const statusConfig: Record<string, { color: string; bg: string; icon: typeof CheckCircle }> = {
  healthy: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30 dark:text-green-400", icon: CheckCircle },
  degraded: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", icon: AlertTriangle },
  down: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30 dark:text-red-400", icon: XCircle },
};

export default function SIEMHealthPage() {
  const t = useTranslations();
  const [destinations, setDestinations] = useState<Destination[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/siem-health", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setDestinations(data.destinations || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const overall = destinations.some((d) => d.status === "down") ? "critical" : destinations.some((d) => d.status === "degraded") ? "degraded" : destinations.length > 0 ? "healthy" : "unknown";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> {t("siemHealth.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("siemHealth.subtitle")}</p>
      </div>

      {overall !== "unknown" && (
        <div className={`rounded-lg border-2 p-4 flex items-center gap-3 ${overall === "healthy" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : overall === "degraded" ? "border-yellow-300 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20" : "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20"}`}>
          {overall === "healthy" ? <CheckCircle className="w-8 h-8 text-green-500" /> : overall === "degraded" ? <AlertTriangle className="w-8 h-8 text-yellow-500" /> : <XCircle className="w-8 h-8 text-red-500" />}
          <div><span className="text-sm text-gray-500">{t("siemHealth.overallStatus")}</span><p className={`text-lg font-bold capitalize ${overall === "healthy" ? "text-green-600" : overall === "degraded" ? "text-yellow-600" : "text-red-600"}`}>{overall}{overall === "degraded" && t("siemHealth.someDegraded")}{overall === "critical" && t("siemHealth.destinationsDown")}</p></div>
        </div>
      )}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("siemHealth.destination")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.endpoint")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.connected")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.latency")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.throughput")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.errorRate")}</th><th className="px-4 py-3 text-left font-medium">{t("siemHealth.lastSuccess")}</th><th className="px-4 py-3 text-left font-medium">{t("common.status")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{destinations.map((d) => { const cfg = statusConfig[d.status]; const Icon = cfg.icon; return (<tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="flex items-center gap-1"><Server className="w-3.5 h-3.5 text-gray-400" /><span className="font-medium">{d.name}</span></span></td><td className="px-4 py-3 font-mono text-xs text-gray-500">{d.endpoint}</td><td className="px-4 py-3">{d.connectivity ? <CheckCircle className="w-4 h-4 text-green-500" /> : <XCircle className="w-4 h-4 text-red-500" />}</td><td className="px-4 py-3"><span className={`text-xs font-medium ${d.latency_ms > 500 ? "text-red-600" : d.latency_ms > 200 ? "text-yellow-600" : "text-green-600"}`}>{d.latency_ms}ms</span></td><td className="px-4 py-3"><span className="flex items-center gap-1 text-xs"><Zap className="w-3 h-3 text-gray-400" />{d.throughput_per_sec}/s</span></td><td className="px-4 py-3"><span className={`text-xs font-medium ${d.error_rate > 5 ? "text-red-600" : d.error_rate > 1 ? "text-yellow-600" : "text-green-600"}`}>{d.error_rate.toFixed(2)}%</span></td><td className="px-4 py-3 text-xs text-gray-400">{d.last_success}</td><td className="px-4 py-3"><span className={`flex items-center gap-1 text-xs ${cfg.color}`}><Icon className="w-3.5 h-3.5" /> {d.status}</span></td></tr>); })}{destinations.length === 0 && !loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">{t("siemHealth.noDestinations")}</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}
