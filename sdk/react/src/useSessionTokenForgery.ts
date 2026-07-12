import { useState, useCallback, useEffect } from "react";

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
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
  return { data, loading, error, refresh: fetchData };
}
