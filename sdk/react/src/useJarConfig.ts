import { useState, useCallback } from "react";

export interface JarPerClient {
  client_id: string;
  client_name: string;
  signing_alg: string;
  require_jar: boolean;
}

export interface JarUsageStats {
  total_requests: number;
  with_jar: number;
  without_jar: number;
  rejected: number;
  by_alg: Record<string, number>;
}

export interface JarConfig {
  require_jar: boolean;
  jar_lifetime_seconds: number;
  signing_alg: "RS256" | "ES256" | "PS256";
  per_client_override: JarPerClient[];
  request_object_preview: string;
  encryption_optional: boolean;
  usage_stats: JarUsageStats;
}

export function useJarConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<JarConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/jar-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: JarConfig = await res.json();
      setConfig(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<JarConfig>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/jar-config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: JarConfig = await res.json();
      setConfig(data);
      return data;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
