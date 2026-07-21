import { useState, useCallback } from "react";

export interface ExposureData {
  user_id: string;
  username: string;
  exposure_score: number;
  active_tokens: number;
  active_sessions: number;
  linked_providers: { provider: string; connected_at: string }[];
  api_keys: { id: string; name: string; last_used: string; scopes: string[] }[];
  recommendations: string[];
}

export function useCredentialExposure(baseUrl: string = "") {
  const [data, setData] = useState<ExposureData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchExposure = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/credential-exposure?user=${encodeURIComponent(user)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: ExposureData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchExposure };
}
