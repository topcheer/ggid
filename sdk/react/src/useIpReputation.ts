import { useState, useCallback } from "react";
export interface IpInfo { ip: string; reputation_score: number; threat_tags: string[]; first_seen: string; last_seen: string; country: string; city: string; isp: string; associated_events: number; blacklisted: boolean; }
export function useIpReputation(baseUrl: string = "") {
  const [info, setInfo] = useState<IpInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const search = useCallback(async (ip: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/ip-reputation?ip=" + encodeURIComponent(ip)); if (!res.ok) throw new Error("HTTP " + res.status); setInfo(await res.json()); } catch (e: any) { setError(e.message); setInfo(null); } finally { setLoading(false); } }, [baseUrl]);
  return { info, loading, error, search };
}
