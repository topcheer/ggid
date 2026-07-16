import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface AutomationRule {
  trigger: string;
  action: string;
  delay_hours: number;
  enabled: boolean;
}

export interface PendingAction {
  user: string;
  action: string;
  trigger_reason: string;
  scheduled_at: string;
}

export interface IdentityDeprovisioningAutomationData {
  automation_rules: AutomationRule[];
  pending_actions: PendingAction[];
  dry_run: boolean;
  success_rate_pct: number;
  processed_7d: number;
}

export function useIdentityDeprovisioningAutomation() {
  const [data, setData] = useState<IdentityDeprovisioningAutomationData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        automation_rules: [
          { trigger: "last_login_days", action: "disable", delay_hours: 24, enabled: true },
          { trigger: "manager_request", action: "revoke", delay_hours: 0, enabled: true },
          { trigger: "hr_system", action: "archive", delay_hours: 72, enabled: true },
          { trigger: "contract_end", action: "revoke", delay_hours: 0, enabled: true },
        ],
        pending_actions: [
          { user: "alice.chen", action: "disable", trigger_reason: "last_login_90d", scheduled_at: "in 2h" },
          { user: "bob.martinez", action: "revoke", trigger_reason: "contract_end", scheduled_at: "in 30m" },
          { user: "carol.jones", action: "archive", trigger_reason: "hr_terminated", scheduled_at: "in 48h" },
        ],
        dry_run: false,
        success_rate_pct: 98,
        processed_7d: 14,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}
