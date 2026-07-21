import { useState, useCallback } from "react";

export interface TokenReuse {
  id: string;
  token_masked: string;
  user_id: string;
  username: string;
  ip_address: string;
  country: string;
  user_agent: string;
  first_seen: string;
  last_seen: string;
  reuse_count: number;
  risk_level: "low" | "medium" | "high" | "critical";
}

export function useTokenReuse(baseUrl: string = "") {
  const [reuses, setReuses] = useState<TokenReuse[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchReuses = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/token-reuse`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setReuses(data.reuses || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { reuses, loading, error, fetchReuses };
}
