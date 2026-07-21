import { useState, useCallback } from "react";

export interface StuffingStats {
  total_attempts: number;
  blocked_by_rate_limit: number;
  blocked_by_captcha: number;
  unique_targeted_accounts: number;
  top_source_ips: { ip: string; attempts: number }[];
  top_user_agents: { ua: string; attempts: number }[];
  attack_pattern: "distributed" | "burst" | "credential_list";
}

export function useCredentialStuffingStats(baseUrl: string = "") {
  const [data, setData] = useState<StuffingStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/credential-stuffing-stats");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
