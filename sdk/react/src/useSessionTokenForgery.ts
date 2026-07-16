import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ForgedToken {
  token: string;
  detection_method: string;
  user_claimed: string;
  actual_source: string;
  timestamp: string;
}

export interface DetectionStats {
  blocked_attempts_24h: number;
  token_validation_failures: number;
  detection_rate_pct: number;
}

export interface SessionTokenForgeryData {
  forged_tokens: ForgedToken[];
  detection_stats: DetectionStats;
}

export function useSessionTokenForgery() {
  const [data, setData] = useState<SessionTokenForgeryData | null>(null);
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
        forged_tokens: [
          { token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsImlzc", detection_method: "signature_invalid", user_claimed: "admin@ggid.dev", actual_source: "203.0.113.50", timestamp: "10m ago" },
          { token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyIiwib3J", detection_method: "claim_mismatch", user_claimed: "user@ggid.dev", actual_source: "198.51.100.22", timestamp: "30m ago" },
          { token: "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJndWVzdCIsIm5hb", detection_method: "issuer_unknown", user_claimed: "guest@ggid.dev", actual_source: "203.0.113.99", timestamp: "1h ago" },
        ],
        detection_stats: {
          blocked_attempts_24h: 47,
          token_validation_failures: 156,
          detection_rate_pct: 99,
        },
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
