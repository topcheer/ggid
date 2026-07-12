import { useState, useCallback, useEffect } from "react";

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
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
  return { data, loading, error, refresh: fetchData };
}
