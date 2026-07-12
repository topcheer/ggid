import { useState, useCallback } from "react";
export interface SsoResult { steps: { name: string; duration_ms: number; status: string; detail?: string }[]; total_ms: number; success: boolean; request_url: string; response_status: number; response_body: string; error?: string; }
export function useSsoDebugger(baseUrl: string = "") {
  const [result, setResult] = useState<SsoResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const trace = useCallback(async (protocol: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/sso-debug?protocol=" + protocol, { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); setResult(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { result, loading, error, trace };
}
