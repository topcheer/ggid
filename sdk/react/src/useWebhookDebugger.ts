import { useState, useCallback } from "react";
export interface DeliveryLog { id: string; timestamp: string; status: number; latency_ms: number; success: boolean; }
export interface TestResult { status: number; status_text: string; headers: Record<string, string>; body: string; latency_ms: number; }
export function useWebhookDebugger(baseUrl: string = "") {
  const [result, setResult] = useState<TestResult | null>(null);
  const [logs, setLogs] = useState<DeliveryLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const sendTest = useCallback(async (endpointId: string, payload: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/webhooks/endpoints/" + endpointId + "/test", { method: "POST", headers: { "Content-Type": "application/json" }, body: payload }); if (!res.ok) throw new Error("HTTP " + res.status); setResult(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const fetchLogs = useCallback(async (endpointId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/webhooks/endpoints/" + endpointId + "/deliveries"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setLogs(data.deliveries || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const replay = useCallback(async (deliveryId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/webhooks/deliveries/" + deliveryId + "/replay", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { result, logs, loading, error, sendTest, fetchLogs, replay };
}
