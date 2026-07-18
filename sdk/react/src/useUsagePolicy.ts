import { useState, useEffect, useCallback } from "react";

export interface ClientUsagePolicy {
  client_id: string;
  client_name: string;
  max_tokens_per_day: number;
  max_sessions: number;
  allowed_ip_ranges: string[];
  enabled: boolean;
}

export function useUsagePolicy(baseUrl: string = "") {
  const [policies, setPolicies] = useState<ClientUsagePolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPolicies = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/usage-policy`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setPolicies(data.policies || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updatePolicy = useCallback(async (clientId: string, policy: Partial<ClientUsagePolicy>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/usage-policy/${clientId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(policy),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setPolicies((prev: any) => prev.map((p) => p.client_id === clientId ? { ...p, ...policy } : p));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  return { policies, loading, error, fetchPolicies, updatePolicy };
}
