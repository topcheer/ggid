import { useState, useCallback } from "react";

export interface TrafficPolicy {
  source: string;
  destination: string;
  policy: "allow" | "deny" | "restrict";
}

export interface IdentityPropagation {
  header_name: string;
  format: "jwt" | "plain";
  propagate: boolean;
}

export interface CircuitBreakingConfig {
  enabled: boolean;
  max_connections: number;
  max_pending_requests: number;
  max_retries: number;
}

export interface ServiceMeshConfig {
  mesh_type: "istio" | "linkerd" | "none";
  mtls_mode: "strict" | "permissive" | "disable";
  traffic_policies: TrafficPolicy[];
  identity_propagation: IdentityPropagation;
  circuit_breaking: CircuitBreakingConfig;
  observability_export: { enabled: boolean; endpoint: string; interval_seconds: number };
}

export function useServiceMeshConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ServiceMeshConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/service-mesh-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<ServiceMeshConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/service-mesh-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
