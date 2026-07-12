import { useState, useCallback } from "react";
export interface HashChainInfo { chain_status: "intact" | "broken"; last_verified_at: string; total_blocks: number; integrity_score: number; tamper_alerts: { block_num: number; expected_hash: string; actual_hash: string; detected_at: string }[]; verify_log: { timestamp: string; status: string; blocks_checked: number }[]; }
export function useHashChain(baseUrl: string = "") {
  const [data, setData] = useState<HashChainInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchChain = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/hash-chain"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const verify = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/hash-chain/verify", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchChain, verify };
}
