import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface TokenFamily {
  family_id: string;
  root_token_hash: string;
  child_count: number;
  status: string;
}

export interface ReuseAlert {
  id: string;
  family_id: string;
  description: string;
  detected_at: string;
}

export interface TokenFamilyAnalysisData {
  families: TokenFamily[];
  reuse_alerts: ReuseAlert[];
}

export function useTokenFamilyAnalysis() {
  const [data, setData] = useState<TokenFamilyAnalysisData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        families: [
          { family_id: "fam_001", root_token_hash: "sha256:abc123...", child_count: 3, status: "active" },
          { family_id: "fam_002", root_token_hash: "sha256:def456...", child_count: 1, status: "active" },
          { family_id: "fam_003", root_token_hash: "sha256:ghi789...", child_count: 5, status: "revoked" },
        ],
        reuse_alerts: [
          { id: "ra1", family_id: "fam_003", description: "Refresh token used after revocation from different IP", detected_at: "2h ago" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
