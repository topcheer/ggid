import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface BreakGlassRole {
  role: string;
  justification_required: boolean;
  auto_expire_minutes: number;
  notify_on_use: boolean;
}

export interface ActiveSession {
  id: string;
  user: string;
  role: string;
  expires_at: string;
  justification: string;
}

export interface UsageRecord {
  user: string;
  role: string;
  timestamp: string;
  duration_minutes: number;
  outcome: string;
}

export interface PolicyBreakGlassData {
  break_glass_roles: BreakGlassRole[];
  active_sessions: ActiveSession[];
  usage_history: UsageRecord[];
  cooldown_period_minutes: number;
  max_concurrent: number;
}

export function usePolicyBreakGlass() {
  const [data, setData] = useState<PolicyBreakGlassData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        break_glass_roles: [
          { role: "Emergency Admin", justification_required: true, auto_expire_minutes: 30, notify_on_use: true },
          { role: "Security Responder", justification_required: true, auto_expire_minutes: 60, notify_on_use: true },
          { role: "DevOps Emergency", justification_required: false, auto_expire_minutes: 15, notify_on_use: false },
        ],
        active_sessions: [
          { id: "bg-1", user: "alice.chen", role: "Emergency Admin", expires_at: "in 12m", justification: "Production DB outage - need admin to restart" },
        ],
        usage_history: [
          { user: "alice.chen", role: "Emergency Admin", timestamp: "2h ago", duration_minutes: 30, outcome: "expired" },
          { user: "bob.martinez", role: "Security Responder", timestamp: "1d ago", duration_minutes: 45, outcome: "revoked" },
          { user: "carol.jones", role: "DevOps Emergency", timestamp: "3d ago", duration_minutes: 15, outcome: "completed" },
          { user: "dave.wilson", role: "Emergency Admin", timestamp: "5d ago", duration_minutes: 30, outcome: "expired" },
        ],
        cooldown_period_minutes: 60,
        max_concurrent: 2,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const activateBreakGlass = useCallback(async (_role: string, _justification: string, _duration: number) => {
    console.log("Activating break glass:", _role);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, activateBreakGlass };
}
