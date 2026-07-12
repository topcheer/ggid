import { useState, useCallback, useEffect } from "react";

export interface TopRisk {
  risk: string;
  category: string;
  score: number;
  status: string;
}

export interface SecurityMetricsDashData {
  mttd_minutes: number;
  mttr_hours: number;
  open_vulns: number;
  patch_compliance_pct: number;
  incidents_30d: number[];
  sla_breaches: number;
  top_10_risks: TopRisk[];
}

export function useSecurityMetricsDash() {
  const [data, setData] = useState<SecurityMetricsDashData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        mttd_minutes: 8,
        mttr_hours: 4,
        open_vulns: 23,
        patch_compliance_pct: 94,
        incidents_30d: [2, 1, 3, 0, 1, 2, 4, 1, 0, 2, 1, 3, 2, 0, 1, 2, 1, 0, 1, 3, 2, 1, 0, 1, 2, 1, 3, 0, 1, 2],
        sla_breaches: 3,
        top_10_risks: [
          { risk: "Missing mTLS between services", category: "Infrastructure", score: 9, status: "mitigated" },
          { risk: "OAuth introspection no auth", category: "Access Control", score: 8, status: "open" },
          { risk: "Webhook SSRF exposure", category: "Data Protection", score: 7, status: "mitigated" },
          { risk: "No audit hash chain", category: "Audit", score: 6, status: "open" },
          { risk: "Password pepper not enforced", category: "Identity", score: 5, status: "mitigated" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
