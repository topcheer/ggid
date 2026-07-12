import { useState, useCallback } from "react";

export interface PolicyException {
  id: string;
  policy_id: string;
  policy_name: string;
  reason: string;
  granted_to: string;
  approver: string;
  risk_override_level: "low" | "medium" | "high" | "critical";
  created_at: string;
  expires_at: string;
  days_remaining: number;
  audit_trail: { timestamp: string; action: string; actor: string; detail: string }[];
}

export function usePolicyExceptions(baseUrl: string = "") {
  const [exceptions, setExceptions] = useState<PolicyException[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchExceptions = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/exceptions`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setExceptions(data.exceptions || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const createException = useCallback(async (payload: { policy_id: string; reason: string; granted_to: string; risk_override_level: string; expires_at: string }) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/exceptions`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { exceptions, loading, error, fetchExceptions, createException };
}
