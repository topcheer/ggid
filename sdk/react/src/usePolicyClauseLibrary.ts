import { useState, useCallback, useEffect } from "react";

export interface Clause {
  clause_id: string;
  category: string;
  text: string;
  parameters: Record<string, string>;
  version: string;
  used_in_policies: string[];
  status: string;
}

export interface PolicyClauseLibraryData {
  clauses: Clause[];
}

export function usePolicyClauseLibrary() {
  const [data, setData] = useState<PolicyClauseLibraryData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        clauses: [
          { clause_id: "AC-001", category: "access_control", text: "Access to production systems requires MFA authentication.", parameters: { mfa_type: "webauthn" }, version: "2.1", used_in_policies: ["prod-access", "admin-access"], status: "active" },
          { clause_id: "AC-002", category: "access_control", text: "Privileged accounts must be reviewed quarterly.", parameters: { review_cycle: "quarterly" }, version: "1.3", used_in_policies: ["privilege-review"], status: "active" },
          { clause_id: "DP-001", category: "data_protection", text: "PII data must be encrypted at rest using AES-256.", parameters: { algorithm: "AES-256-GCM" }, version: "3.0", used_in_policies: ["pii-handling", "data-classification"], status: "active" },
          { clause_id: "DP-002", category: "data_protection", text: "Data retention period for audit logs is 7 years.", parameters: { retention_years: "7" }, version: "1.0", used_in_policies: ["audit-retention"], status: "active" },
          { clause_id: "AU-001", category: "audit", text: "All authentication events must be logged with timestamp, IP, and user agent.", parameters: {}, version: "1.2", used_in_policies: ["auth-audit", "siem-forward"], status: "active" },
          { clause_id: "AU-002", category: "audit", text: "Audit logs must be tamper-evident using hash chaining.", parameters: { hash_algo: "SHA-256" }, version: "2.0", used_in_policies: ["audit-integrity"], status: "active" },
          { clause_id: "CO-001", category: "compliance", text: "Access reviews must be completed within 30 days of trigger.", parameters: { deadline_days: "30" }, version: "1.1", used_in_policies: ["access-review", "soc2-audit"], status: "active" },
          { clause_id: "CO-002", category: "compliance", text: "GDPR data subject requests must be processed within 30 days.", parameters: { sla_days: "30" }, version: "2.0", used_in_policies: ["gdpr-dsr"], status: "active" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
