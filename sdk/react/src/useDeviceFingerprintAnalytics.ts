import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface FingerprintCluster {
  browser: string;
  os: string;
  device_count: number;
  pct: number;
}

export interface SuspiciousFingerprint {
  fingerprint_hash: string;
  reason: string;
  associated_user: string;
  timestamp: string;
}

export interface KnownGood {
  hash: string;
  last_seen: string;
}

export interface DeviceFingerprintAnalyticsData {
  unique_fingerprints: number;
  fingerprint_clusters: FingerprintCluster[];
  known_good_list: KnownGood[];
  suspicious_fingerprints: SuspiciousFingerprint[];
  fingerprint_match_rate_pct: number;
  canvas_hash_distribution: number[];
  webgl_hash_distribution: number[];
}

export function useDeviceFingerprintAnalytics() {
  const [data, setData] = useState<DeviceFingerprintAnalyticsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
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
      setData({
        unique_fingerprints: 15420,
        fingerprint_clusters: [
          { browser: "Chrome", os: "Windows", device_count: 5800, pct: 38 },
          { browser: "Safari", os: "iOS", device_count: 3200, pct: 21 },
          { browser: "Chrome", os: "macOS", device_count: 2400, pct: 16 },
          { browser: "Chrome", os: "Android", device_count: 1800, pct: 12 },
          { browser: "Firefox", os: "Linux", device_count: 1220, pct: 8 },
          { browser: "Edge", os: "Windows", device_count: 1000, pct: 5 },
        ],
        known_good_list: [
          { hash: "a1b2c3d4e5f6789a", last_seen: "1m ago" },
          { hash: "b2c3d4e5f6789a1b", last_seen: "5m ago" },
          { hash: "c3d4e5f6789a1b2c", last_seen: "10m ago" },
          { hash: "d4e5f6789a1b2c3d", last_seen: "20m ago" },
          { hash: "e5f6789a1b2c3d4e", last_seen: "1h ago" },
          { hash: "f6789a1b2c3d4e5f", last_seen: "2h ago" },
        ],
        suspicious_fingerprints: [
          { fingerprint_hash: "9f8e7d6c5b4a3928", reason: "headless_browser", associated_user: "unknown", timestamp: "5m ago" },
          { fingerprint_hash: "1a2b3c4d5e6f7890", reason: "spoofed", associated_user: "user.123@temp.com", timestamp: "15m ago" },
          { fingerprint_hash: "0f9e8d7c6b5a4938", reason: "inconsistent", associated_user: "suspicious@mail.com", timestamp: "30m ago" },
        ],
        fingerprint_match_rate_pct: 94,
        canvas_hash_distribution: [5, 12, 28, 45, 67, 89, 72, 45, 23, 15, 8, 4],
        webgl_hash_distribution: [8, 15, 22, 38, 55, 78, 91, 65, 41, 22, 10, 5],
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
