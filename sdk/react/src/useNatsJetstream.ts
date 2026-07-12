import { useState, useCallback } from "react";
export interface JetStreamData { streams: { name: string; subjects: string[]; msgs: number; bytes: number; consumer_count: number; last_msg: string }[]; consumers: { name: string; stream: string; delivered: number; ack_floor: number; pending: number; status: string }[]; connections: number; total_msgs: number; total_bytes: number; throughput_per_sec: number; status: string; }
export function useNatsJetstream(baseUrl: string = "") {
  const [data, setData] = useState<JetStreamData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/nats-health"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
