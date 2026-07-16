import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ClientCoverage {
  client: string;
  bound_pct: number;
  binding_method: string;
  last_checked: string;
}

export interface UnboundToken {
  token_id: string;
  client: string;
  issued_at: string;
}

export interface SessionTokenBindingCoverageData {
  coverage_pct: number;
  bound_tokens: number;
  unbound_tokens: number;
  compliance_threshold: number;
  per_client: ClientCoverage[];
  unbound_list: UnboundToken[];
}

export function useSessionTokenBindingCoverage() {
  const [data, setData] = useState<SessionTokenBindingCoverageData | null>(null);
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
        coverage_pct: 73,
        bound_tokens: 8760,
        unbound_tokens: 3240,
        compliance_threshold: 90,
        per_client: [
          { client: "web-console", bound_pct: 98, binding_method: "DPoP", last_checked: "5m ago" },
          { client: "mobile-app", bound_pct: 85, binding_method: "DPoP", last_checked: "10m ago" },
          { client: "ci-cd-bot", bound_pct: 100, binding_method: "mTLS", last_checked: "2m ago" },
          { client: "legacy-api", bound_pct: 12, binding_method: "none", last_checked: "1h ago" },
        ],
        unbound_list: [
          { token_id: "tok_a1b2c3", client: "legacy-api", issued_at: "2h ago" },
          { token_id: "tok_d4e5f6", client: "legacy-api", issued_at: "3h ago" },
          { token_id: "tok_g7h8i9", client: "mobile-app", issued_at: "30m ago" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
