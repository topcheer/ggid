import { useState, useCallback } from "react";
export interface ClaimConfig { standard_claims: Record<string, boolean>; custom_claims: { name: string; source: string; value: string }[]; scope_mappings: Record<string, string[]>; token_type: string; }
export function useOidcClaimConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ClaimConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/oidc-claim-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: ClaimConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/oidc-claim-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
