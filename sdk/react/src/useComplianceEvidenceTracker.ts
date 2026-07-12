import { useState, useCallback, useEffect } from "react";

export interface EvidenceRow {
  control: string;
  evidence_type: string;
  last_collected: string;
  next_due: string;
  owner: string;
  status: string;
}

export interface OverdueAlert {
  control_id: string;
  framework: string;
  days_overdue: number;
}

export interface AutoCollectionRule {
  rule_name: string;
  description: string;
  enabled: boolean;
}

export interface ComplianceEvidenceTrackerData {
  frameworks: Record<string, EvidenceRow[]>;
  overdue_alerts: OverdueAlert[];
  auto_collection_rules: AutoCollectionRule[];
}

export function useComplianceEvidenceTracker() {
  const [data, setData] = useState<ComplianceEvidenceTrackerData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        frameworks: {
          SOC2: [
            { control: "CC6.1", evidence_type: "Access control policy", last_collected: "5d ago", next_due: "25d", owner: "sec-lead", status: "collected" },
            { control: "CC7.2", evidence_type: "Incident response log", last_collected: "30d ago", next_due: "0d", owner: "sec-ops", status: "overdue" },
            { control: "CC8.1", evidence_type: "Change management records", last_collected: "3d ago", next_due: "27d", owner: "dev-lead", status: "collected" },
          ],
          HIPAA: [
            { control: "164.312(a)(1)", evidence_type: "Access controls audit", last_collected: "60d ago", next_due: "0d", owner: "compliance", status: "overdue" },
            { control: "164.312(b)", evidence_type: "Audit controls report", last_collected: "10d ago", next_due: "20d", owner: "compliance", status: "collected" },
          ],
          ISO27001: [
            { control: "A.9.4.2", evidence_type: "Privileged access matrix", last_collected: "15d ago", next_due: "15d", owner: "iso-lead", status: "pending" },
            { control: "A.12.6.1", evidence_type: "Vulnerability scan results", last_collected: "2d ago", next_due: "28d", owner: "sec-ops", status: "collected" },
          ],
          GDPR: [
            { control: "Art.30", evidence_type: "Processing records", last_collected: "3d ago", next_due: "27d", owner: "dpo", status: "collected" },
            { control: "Art.32", evidence_type: "Security measures doc", last_collected: "8d ago", next_due: "22d", owner: "dpo", status: "collected" },
          ],
          "PCI-DSS": [
            { control: "Req 8.3", evidence_type: "MFA enforcement proof", last_collected: "45d ago", next_due: "15d", owner: "sec-lead", status: "pending" },
          ],
        },
        overdue_alerts: [
          { control_id: "CC7.2", framework: "SOC2", days_overdue: 0 },
          { control_id: "164.312(a)(1)", framework: "HIPAA", days_overdue: 30 },
        ],
        auto_collection_rules: [
          { rule_name: "Auto-collect access logs", description: "Collect access control logs monthly for SOC2 CC6.1", enabled: true },
          { rule_name: "Auto-collect scan results", description: "Collect vulnerability scan results quarterly", enabled: true },
          { rule_name: "Auto-collect incident reports", description: "Collect incident response reports monthly", enabled: false },
          { rule_name: "Auto-collect policy acknowledgments", description: "Collect signed policy acknowledgments annually", enabled: true },
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
