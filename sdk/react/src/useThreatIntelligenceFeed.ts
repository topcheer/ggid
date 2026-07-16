import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface IntelSource {
  source_name: string;
  type: string;
  last_sync: string;
  status: string;
}

export interface FeedHealth {
  source_name: string;
  uptime_pct: number;
  indicators_count: number;
}

export interface ThreatIndicator {
  indicator: string;
  type: string;
  confidence: number;
  first_seen: string;
  last_seen: string;
  source: string;
  tags: string[];
}

export interface AutoBlockRule {
  rule_name: string;
  description: string;
  enabled: boolean;
}

export interface ThreatIntelligenceFeedData {
  intel_sources: IntelSource[];
  feed_health: FeedHealth[];
  indicators: ThreatIndicator[];
  auto_block_rules: AutoBlockRule[];
}

export function useThreatIntelligenceFeed() {
  const [data, setData] = useState<ThreatIntelligenceFeedData | null>(null);
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
        intel_sources: [
          { source_name: "AlienVault OTX", type: "IP/domain/hash", last_sync: "5m ago", status: "active" },
          { source_name: "AbuseIPDB", type: "IP", last_sync: "10m ago", status: "active" },
          { source_name: "VirusTotal", type: "hash", last_sync: "15m ago", status: "active" },
          { source_name: "Have I Been Pwned", type: "email", last_sync: "1h ago", status: "active" },
          { source_name: "PhishTank", type: "domain", last_sync: "2h ago", status: "degraded" },
        ],
        feed_health: [
          { source_name: "AlienVault OTX", uptime_pct: 99.8, indicators_count: 45200 },
          { source_name: "AbuseIPDB", uptime_pct: 99.5, indicators_count: 28100 },
          { source_name: "VirusTotal", uptime_pct: 98.2, indicators_count: 89300 },
          { source_name: "Have I Been Pwned", uptime_pct: 100, indicators_count: 3200 },
          { source_name: "PhishTank", uptime_pct: 87.3, indicators_count: 5600 },
        ],
        indicators: [
          { indicator: "185.220.101.45", type: "IP", confidence: 0.95, first_seen: "3d ago", last_seen: "5m ago", source: "AbuseIPDB", tags: ["tor_exit", "credential_stuffing"] },
          { indicator: "malware-c2.evil.com", type: "domain", confidence: 0.89, first_seen: "1w ago", last_seen: "1h ago", source: "AlienVault OTX", tags: ["c2", "malware"] },
          { indicator: "a1b2c3d4e5f6...", type: "hash", confidence: 0.78, first_seen: "5d ago", last_seen: "2h ago", source: "VirusTotal", tags: ["trojan", "windows"] },
          { indicator: "phishing-login.gotcha.io", type: "domain", confidence: 0.92, first_seen: "2d ago", last_seen: "30m ago", source: "PhishTank", tags: ["phishing", "credential_theft"] },
          { indicator: "45.137.21.89", type: "IP", confidence: 0.71, first_seen: "12h ago", last_seen: "10m ago", source: "AlienVault OTX", tags: ["scanner", "brute_force"] },
        ],
        auto_block_rules: [
          { rule_name: "Block known bad IPs", description: "Auto-block requests from IPs with confidence > 0.8", enabled: true },
          { rule_name: "Block compromised credentials", description: "Check login against HIBP database", enabled: true },
          { rule_name: "Block Tor exit nodes", description: "Block all Tor exit node IPs", enabled: false },
        ],
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
