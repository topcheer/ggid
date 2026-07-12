import { useState, useCallback } from "react";
export interface ApiKey { id: string; name: string; scopes: string[]; created_at: string; expires_at: string; last_used: string | null; status: "active" | "expired" | "revoked"; usage_count: number; }
export function useAPIKeyLifecycle(baseUrl: string = "") {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchKeys = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/api-keys"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setKeys(data.keys || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const createKey = useCallback(async (name: string, scopes: string[], expiresAt: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/api-keys", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name, scopes, expires_at: expiresAt }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const rotateKey = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/api-keys/" + id + "/rotate", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const revokeKey = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/api-keys/" + id, { method: "DELETE" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { keys, loading, error, fetchKeys, createKey, rotateKey, revokeKey };
}
