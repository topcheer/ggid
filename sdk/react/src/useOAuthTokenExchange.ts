import { useState, useCallback } from "react";
export interface ExchangeResult { access_token: string; token_type: string; expires_in: number; scope: string; issued_token_type: string; }
export function useOAuthTokenExchange(baseUrl: string = "") {
  const [result, setResult] = useState<ExchangeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const exchange = useCallback(async (subjectToken: string, actorToken: string, audience: string, scope: string, resource: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/token-exchange", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ grant_type: "urn:ietf:params:oauth:grant-type:token-exchange", subject_token: subjectToken, actor_token: actorToken || undefined, audience, scope, resource }) }); if (!res.ok) throw new Error("HTTP " + res.status); setResult(await res.json()); } catch (e: any) { setError(e.message); setResult(null); } finally { setLoading(false); } }, [baseUrl]);
  return { result, loading, error, exchange };
}
