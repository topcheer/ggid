"use client";

import { useState, useEffect, useCallback } from "react";
import { Globe, MapPin, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface GeoData {
  unique_locations: number;
  countries: { code: string; name: string; sessions: number; pct: number }[];
  top_cities: { city: string; country: string; sessions: number }[];
  risk_geographies: { country: string; reason: string; risk_level: "low" | "medium" | "high" }[];
  heatmap: { lat: number; lng: number; intensity: number; label: string }[];
}

const riskColors: Record<string, string> = {
  low: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function SessionGeoStatsPage() {
  const t = useTranslations();
  const [data, setData] = useState<GeoData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/session-geo-stats", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxCity = Math.max(...(data?.top_cities.map((c: any) => c.sessions) || [1]), 1);
  const maxHeat = Math.max(...(data?.heatmap.map((h: any) => h.intensity) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-cyan-500" /> Session Geo Stats</h1>
        <p className="text-sm text-gray-500 mt-1">{t("sessionGeoStats.subtitle")}</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><MapPin className="w-8 h-8 text-cyan-500" /><div><span className="text-sm text-gray-500">{t("sessionGeoStats.uniqueLocations")}</span><p className="text-xl font-bold">{data.unique_locations}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("sessionGeoStats.countries")}</span><p className="text-xl font-bold mt-1">{data.countries.length}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("sessionGeoStats.topCitySessions")}</span><p className="text-xl font-bold mt-1">{data.top_cities[0]?.sessions || 0}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("sessionGeoStats.riskGeographies")}</span><p className="text-xl font-bold text-red-600 mt-1">{data.risk_geographies.length}</p></div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">{t("sessionGeoStats.topCities")}</h3>
              <div className="space-y-2">{data.top_cities.map((c: any, i: number) => (
                <div key={i} className="flex items-center gap-2"><span className="text-xs text-gray-500 w-32 truncate">{c.city}, {c.country}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5 overflow-hidden"><div className="h-full bg-cyan-500 rounded-full" style={{ width: `${(c.sessions / maxCity) * 100}%` }} /></div><span className="text-xs font-bold w-10 text-right">{c.sessions}</span></div>
              ))}{data.top_cities.length === 0 && <p className="text-xs text-gray-400">{t("sessionGeoStats.noData")}</p>}</div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">{t("sessionGeoStats.countries")}</h3>
              <div className="space-y-1">{data.countries.map((c: any) => (
                <div key={c.code} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">{c.code}</span><span className="flex-1">{c.name}</span><span className="font-bold">{c.sessions}</span><span className="text-xs text-gray-400">{c.pct.toFixed(1)}%</span></div>
              ))}</div>
            </div>
          </div>

          {data.risk_geographies.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-red-500" /> {t("sessionGeoStats.riskGeographies")}</h3>
              <div className="space-y-2">{data.risk_geographies.map((r: any, i: number) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">{r.country}</span><span className="flex-1 text-gray-500">{r.reason}</span><span className={`px-2 py-0.5 rounded text-xs ${riskColors[r.risk_level]}`}>{r.risk_level}</span></div>
              ))}</div>
            </div>
          )}

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">{t("sessionGeoStats.heatmapGrid")}</h3>
            <div className="grid grid-cols-8 gap-1">
              {data.heatmap.map((h: any, i: number) => (
                <div key={i} className="aspect-square rounded" style={{ background: `rgba(239, 68, 68, ${h.intensity / maxHeat})` }} title={`${h.label}: ${h.intensity}`} />
              ))}
              {data.heatmap.length === 0 && <p className="text-xs text-gray-400 col-span-8">{t("sessionGeoStats.noHeatmapData")}</p>}
            </div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("sessionGeoStats.loading")}</p>}
    </div>
  );
}
