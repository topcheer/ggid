import { useState, useCallback } from "react";
export interface GatewayConfig { timeout_seconds: number; max_body_size_kb: number; circuit_breaker_enabled: boolean; retry_max_attempts: number; retry_backoff_ms: number; health_check_interval_seconds: number; cors_origins: string[]; route_limits: { route: string; requests_per_min: number; burst: number }[]; }
export function useApiGatewayConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<GatewayConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/gateway-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: GatewayConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/gateway-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
