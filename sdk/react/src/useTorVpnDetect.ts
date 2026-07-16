import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface DetectedConnection {
  ip: string;
  type: string;
  confidence: number;
  first_seen: string;
  user: string;
}

export interface BlocklistRule {
  rule_name: string;
  enabled: boolean;
}

export interface CountryStat {
  country: string;
  connections: number;
}

export interface TorVpnDetectData {
  detected_connections: DetectedConnection[];
  exit_node_list: string[];
  blocklist_rules: BlocklistRule[];
  per_country_stats: CountryStat[];
  auto_challenge_enabled: boolean;
}

export function useTorVpnDetect() {
  const [data, setData] = useState<TorVpnDetectData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        detected_connections: [
          { ip: "185.220.101.45", type: "tor", confidence: 0.98, first_seen: "5m ago", user: "anonymous login" },
          { ip: "91.213.50.182", type: "tor", confidence: 0.95, first_seen: "15m ago", user: "user.123" },
          { ip: "45.83.221.10", type: "vpn", confidence: 0.82, first_seen: "30m ago", user: "employee.dev" },
          { ip: "196.52.43.134", type: "proxy", confidence: 0.76, first_seen: "1h ago", user: "api.client" },
        ],
        exit_node_list: ["185.220.101.45", "91.213.50.182", "192.42.116.16", "199.249.230.35", "171.25.193.78", "82.221.139.163", "104.244.72.115", "139.99.97.27"],
        blocklist_rules: [
          { rule_name: "Block TOR exit nodes", enabled: true },
          { rule_name: "Challenge VPN connections", enabled: true },
          { rule_name: "Block known proxy services", enabled: false },
        ],
        per_country_stats: [
          { country: "Netherlands", connections: 12 },
          { country: "United States", connections: 8 },
          { country: "Germany", connections: 6 },
          { country: "France", connections: 4 },
          { country: "Russia", connections: 3 },
        ],
        auto_challenge_enabled: false,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
