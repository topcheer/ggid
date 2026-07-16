"use client";

import { useState, useEffect, useCallback } from "react";
import { Plane, Calendar, AlertTriangle, MapPin, Zap, Filter } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TravelAlert {
  id: string;
  user_id: string;
  username: string;
  from_city: string;
  from_country: string;
  to_city: string;
  to_country: string;
  distance_km: number;
  time_gap_minutes: number;
  speed_kmh: number;
  from_ip: string;
  to_ip: string;
  detected_at: string;
  risk_level: "medium" | "high" | "critical";
}

const riskColors: Record<string, string> = {
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ImpossibleTravelPage() {
  const t = useTranslations();

  const [alerts, setAlerts] = useState<TravelAlert[]>([]);
  const [loading, setLoading] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    const params = startDate && endDate ? `?start=${startDate}&end=${endDate}` : "";
    try {
      const res = await fetch(`/api/v1/auth/impossible-travel${params}`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setAlerts(data.alerts || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [startDate, endDate]);

  useEffect(() => {
    const end = new Date(); const start = new Date(); start.setDate(start.getDate() - 7);
    setStartDate(start.toISOString().split("T")[0]); setEndDate(end.toISOString().split("T")[0]);
  }, []);

  useEffect(() => { if (startDate && endDate) fetchData(); }, [startDate, endDate, fetchData]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Plane className="w-6 h-6 text-orange-500" /> {t("big1.impossibleTravel.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("big1.impossibleTravel.detectLoginsFromGeographicallyImpossibleDistancesWithinATimeWindow")}</p>
      </div>

      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <Calendar className="w-4 h-4 text-gray-400" />
          <input aria-label="Start date" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          <span className="text-gray-400">{t("big1.impossibleTravel.to")}</span>
          <input aria-label="End date" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button aria-label="action" onClick={fetchData} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? t("big1.impossibleTravel.loading") : t("big1.impossibleTravel.refresh")}</button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.impossibleTravel.totalAlerts")}</span><p className="text-2xl font-bold mt-1">{alerts.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.impossibleTravel.critical")}</span><p className="text-2xl font-bold mt-1 text-red-600">{alerts.filter((a) => a.risk_level === "critical").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.impossibleTravel.uniqueUsers")}</span><p className="text-2xl font-bold mt-1">{new Set(alerts.map((a) => a.user_id)).size}</p></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.user")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.fromTo")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.distance")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.timeGap")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.speed")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.detected")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.impossibleTravel.risk")}</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {alerts.map((a) => (
              <tr key={a.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium">{a.username}</td>
                <td className="px-4 py-3"><div className="flex items-center gap-1 text-xs"><MapPin className="w-3 h-3 text-gray-400" /><span>{a.from_city}, {a.from_country}</span><span className="text-gray-300">→</span><span>{a.to_city}, {a.to_country}</span></div><div className="flex items-center gap-2 text-xs text-gray-400 mt-0.5"><span className="font-mono">{a.from_ip}</span><span>→</span><span className="font-mono">{a.to_ip}</span></div></td>
                <td className="px-4 py-3 font-bold">{a.distance_km.toLocaleString()}{t("big1.impossibleTravel.km")}</td>
                <td className="px-4 py-3">{a.time_gap_minutes}{t("big1.impossibleTravel.m")}</td>
                <td className="px-4 py-3"><span className="flex items-center gap-1 font-bold text-orange-600"><Zap className="w-3 h-3" />{a.speed_kmh.toLocaleString()}{t("big1.impossibleTravel.kmH")}</span></td>
                <td className="px-4 py-3 text-gray-500">{a.detected_at}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[a.risk_level]}`}>{a.risk_level}</span></td>
              </tr>
            ))}
            {alerts.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">{t("big1.impossibleTravel.noImpossibleTravelAlerts")}</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
