import { useState, useCallback } from "react";

export interface ChainEntry {
  id: string;
  step: number;
  action: string;
  collected_by: string;
  collected_at: string;
  hash: string;
  prev_hash: string;
  verified_by: string | null;
  verified_at: string | null;
  status: "pending" | "verified" | "failed";
  evidence_type: string;
  description: string;
}

export interface EvidenceChain {
  control_id: string;
  control_name: string;
  framework: string;
  entries: ChainEntry[];
  chain_intact: boolean;
}

export function useEvidenceChain(baseUrl: string = "") {
  const [chain, setChain] = useState<EvidenceChain | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchChain = useCallback(async (controlId: string) => {
    if (!controlId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/evidence-chain?control_id=${encodeURIComponent(controlId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: EvidenceChain = await res.json();
      setChain(data);
    } catch (e: any) {
      setError(e.message);
      setChain(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const verifyEntry = useCallback(async (entryId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/evidence-chain/${entryId}/verify`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setChain((prev: any) => prev ? {
        ...prev,
        entries: prev.entries.map((e) => e.id === entryId ? { ...e, status: "verified", verified_by: "current_user", verified_at: new Date().toISOString() } : e),
      } : null);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { chain, loading, error, fetchChain, verifyEntry };
}
