import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface GapSeverityBreakdown {
  critical: number;
  high: number;
  medium: number;
  low: number;
}

export interface ComplianceGap {
  framework: string;
  control_id: string;
  requirement: string;
  current_state: string;
  gap_severity: "critical" | "high" | "medium" | "low";
  remediation_plan: string;
  owner: string;
  deadline: string;
}

export interface ComplianceGapReportData {
  frameworks: string[];
  gaps: ComplianceGap[];
  summary: {
    total_gaps: number;
    by_severity: GapSeverityBreakdown;
    resolved_30d: number;
  };
}

export function useComplianceGapReport() {
  const [data, setData] = useState<ComplianceGapReportData | null>(null);
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
        frameworks: ["SOC2", "ISO 27001", "GDPR", "HIPAA", "PCI-DSS"],
        gaps: [
          { framework: "SOC2", control_id: "CC6.1", requirement: "Logical access controls", current_state: "Partially implemented", gap_severity: "high", remediation_plan: "Deploy MFA enforcement for all admin accounts", owner: "Security Team", deadline: "2026-02-15" },
          { framework: "SOC2", control_id: "CC7.2", requirement: "System monitoring", current_state: "Missing SIEM integration", gap_severity: "critical", remediation_plan: "Complete SIEM forwarding for all services", owner: "DevOps", deadline: "2026-01-30" },
          { framework: "SOC2", control_id: "CC8.1", requirement: "Change management", current_state: "Documented", gap_severity: "low", remediation_plan: "Add automated approval workflow", owner: "Engineering", deadline: "2026-03-01" },
          { framework: "ISO 27001", control_id: "A.9.2.3", requirement: "User access management", current_state: "Manual review", gap_severity: "medium", remediation_plan: "Implement quarterly access review automation", owner: "IT Admin", deadline: "2026-04-15" },
          { framework: "ISO 27001", control_id: "A.12.4", requirement: "Event logging", current_state: "Implemented", gap_severity: "low", remediation_plan: "Extend log retention to 365 days", owner: "DevOps", deadline: "2026-02-28" },
          { framework: "GDPR", control_id: "Art.32", requirement: "Data protection measures", current_state: "Encryption in transit", gap_severity: "high", remediation_plan: "Add field-level encryption for PII at rest", owner: "Data Team", deadline: "2026-01-20" },
          { framework: "HIPAA", control_id: "164.312(a)(1)", requirement: "Access control", current_state: "RBAC implemented", gap_severity: "medium", remediation_plan: "Add ABAC for sensitive records", owner: "Security Team", deadline: "2026-03-10" },
          { framework: "PCI-DSS", control_id: "Req-8", requirement: "Authentication", current_state: "MFA partial", gap_severity: "critical", remediation_plan: "Enforce MFA for all cardholder data access", owner: "Security Team", deadline: "2026-01-15" },
        ],
        summary: {
          total_gaps: 8,
          by_severity: { critical: 2, high: 2, medium: 2, low: 2 },
          resolved_30d: 5,
        },
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
