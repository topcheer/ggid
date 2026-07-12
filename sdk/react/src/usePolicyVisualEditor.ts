import { useState, useCallback, useEffect } from "react";

export interface PolicyNode {
  id: string;
  type: "subject" | "condition" | "action";
  label: string;
  properties: Record<string, string>;
}

export interface PolicyTemplate {
  name: string;
  description: string;
}

export interface PolicyVisualEditorData {
  nodes: PolicyNode[];
  template_gallery: PolicyTemplate[];
}

export function usePolicyVisualEditor() {
  const [data, setData] = useState<PolicyVisualEditorData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        nodes: [
          { id: "s1", type: "subject", label: "Role: Developer", properties: { role: "developer", source: "any" } },
          { id: "s2", type: "subject", label: "Role: Admin", properties: { role: "admin", source: "corporate" } },
          { id: "c1", type: "condition", label: "Business Hours", properties: { time: "08:00-18:00", days: "Mon-Fri", tz: "UTC" } },
          { id: "c2", type: "condition", label: "IP in Corporate Range", properties: { cidr: "10.0.0.0/8" } },
          { id: "a1", type: "action", label: "Allow Access", properties: { decision: "allow", mfa: "not_required" } },
          { id: "a2", type: "action", label: "Require MFA", properties: { decision: "allow", mfa: "required", step_up: "webauthn" } },
        ],
        template_gallery: [
          { name: "MFA for Admins", description: "Require WebAuthn for admin role" },
          { name: "Business Hours Only", description: "Block access outside working hours" },
          { name: "Corporate Network", description: "Require corporate IP" },
          { name: "Geo-Fenced Access", description: "Restrict by country" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const validateFlow = useCallback(async () => {
    console.log("Validating policy flow");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, validateFlow };
}
