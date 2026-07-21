import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ConflictingRolePair {
  role_a: string;
  role_b: string;
  conflict_type: "creation" | "approval" | "access";
}

export interface SodViolation {
  user: string;
  conflicting_roles: string[];
  detected_at: string;
  action_required: string;
}

export interface PolicySoDConfigData {
  conflicting_roles: ConflictingRolePair[];
  sod_violations: SodViolation[];
  auto_enforce: boolean;
  bypass_requires_c_level: boolean;
}

export function usePolicySoDConfig() {
  const [data, setData] = useState<PolicySoDConfigData | null>(null);
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
        conflicting_roles: [
          { role_a: "Payment Initiator", role_b: "Payment Approver", conflict_type: "approval" },
          { role_a: "User Creator", role_b: "Access Assigner", conflict_type: "creation" },
          { role_a: "Admin", role_b: "Audit Reviewer", conflict_type: "access" },
          { role_a: "Config Editor", role_b: "Config Deployer", conflict_type: "approval" },
        ],
        sod_violations: [
          { user: "alice.chen", conflicting_roles: ["Payment Initiator", "Payment Approver"], detected_at: "2h ago", action_required: "immediate" },
          { user: "bob.martinez", conflicting_roles: ["Admin", "Audit Reviewer"], detected_at: "1d ago", action_required: "review" },
        ],
        auto_enforce: false,
        bypass_requires_c_level: true,
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
