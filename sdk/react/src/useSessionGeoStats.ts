import { useState, useCallback } from "react";

export interface GeoData {
  unique_locations: number;
  countries: { code: string; name: string; sessions: number; pct: number }[];
  top_cities: { city: string; country: string; sessions: number }[];
  risk_geographies: { country: string; reason: string; risk_level: "low" | "medium" | "high" }[];
  heatmap: { lat: number; lng: number; intensity: number; label: string }[];
}

export function useSessionGeoStats(baseUrl: string = "") {
  const [data, setData] = useState<GeoData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGeo = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/session-geo-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchGeo };
}
