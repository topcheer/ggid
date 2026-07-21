import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface GeoRule {
  id: string;
  country: string;
  cidr: string;
  action: string;
  label: string;
}

export interface GeoFencingConfigData {
  enabled: boolean;
  rules: GeoRule[];
  whitelist_ips: string[];
}

export function useGeoFencingConfig() {
  const [data, setData] = useState<GeoFencingConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({ enabled: true,
        rules: [
          { id: "r1", country: "US", cidr: "0.0.0.0/0", action: "allow", label: "Corporate VPN" },
          { id: "r2", country: "CN", cidr: "0.0.0.0/0", action: "deny", label: "Blocked region" },
          { id: "r3", country: "RU", cidr: "0.0.0.0/0", action: "challenge", label: "Step-up required" },
        ],
        whitelist_ips: ["10.0.0.0/8", "172.16.0.0/12", "192.168.1.100"], });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
