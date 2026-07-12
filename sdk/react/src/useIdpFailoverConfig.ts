import { useState, useCallback, useEffect } from "react";

export interface IdpCard {
  name: string;
  role: string;
  status: string;
  latency_ms: number;
  health_score: number;
}

export interface FailoverRule {
  trigger: string;
  condition: string;
  action: string;
}

export interface FailoverHistoryEntry {
  id: string;
  timestamp: string;
  from: string;
  to: string;
  reason: string;
}

export interface IdpFailoverConfigData {
  idp_cards: IdpCard[];
  failover_rules: FailoverRule[];
  failover_history: FailoverHistoryEntry[];
  health_check_interval: string;
  auto_fallback: boolean;
}

export function useIdpFailoverConfig() {
  const [data, setData] = useState<IdpFailoverConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        idp_cards: [
          { name: "Azure AD (Primary)", role: "primary", status: "healthy", latency_ms: 45, health_score: 98 },
          { name: "Okta (Secondary)", role: "secondary", status: "healthy", latency_ms: 120, health_score: 95 },
        ],
        failover_rules: [
          { trigger: "Latency Threshold", condition: "> 500ms for 3 consecutive checks", action: "Switch to secondary" },
          { trigger: "Error Rate", condition: "> 10% in 5min window", action: "Switch to secondary" },
          { trigger: "Unreachable", condition: "3 consecutive health check failures", action: "Immediate switch" },
        ],
        failover_history: [
          { id: "1", timestamp: "3d ago", from: "Azure AD", to: "Okta", reason: "Latency > 500ms" },
          { id: "2", timestamp: "3d ago", from: "Okta", to: "Azure AD", reason: "Primary recovered" },
        ],
        health_check_interval: "30s",
        auto_fallback: true,
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const manualSwitch = useCallback((idpName: string) => { console.log("Manual switch to", idpName); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, manualSwitch };
}
