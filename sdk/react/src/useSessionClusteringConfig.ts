import { useState, useCallback } from "react";

export interface RedisNode {
  host: string;
  port: number;
  role: "master" | "replica";
}

export interface SessionClusteringConfig {
  cluster_topology: "single" | "HA" | "cluster";
  redis_nodes: RedisNode[];
  partition_strategy: "by_tenant" | "by_user";
  eviction_policy: "lru" | "lfu" | "ttl";
  failover_mode: "automatic" | "manual";
  serialization_format: "json" | "msgpack" | "protobuf";
}

export function useSessionClusteringConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<SessionClusteringConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/session-clustering-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<SessionClusteringConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/session-clustering-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
