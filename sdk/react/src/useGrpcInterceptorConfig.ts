import { useState, useCallback } from "react";

export interface InterceptorEntry {
  name: string;
  order: number;
  enabled: boolean;
  latency_ms: number;
}

export interface GrpcInterceptorConfig {
  interceptor_chain: InterceptorEntry[];
}

export function useGrpcInterceptorConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<GrpcInterceptorConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/grpc-interceptor-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<GrpcInterceptorConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/grpc-interceptor-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  const testInterceptor = useCallback(async (name: string) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/grpc-interceptor-config/test`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name }) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); return await res.json(); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig, testInterceptor };
}
