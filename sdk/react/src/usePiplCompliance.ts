import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface DataHandlingRules {
  consent_required: boolean;
  data_minimization: boolean;
  purpose_limitation: boolean;
  cross_border_assessment: boolean;
}

export interface ConsentLogEntry {
  user: string;
  purpose: string;
  timestamp: string;
  withdrawn: boolean;
}

export interface DpoInfo {
  name: string;
  email: string;
}

export interface CrossBorderApplication {
  applicant: string;
  data_type: string;
  recipient_country: string;
  status: string;
  assessment_result: string;
}

export interface RetentionItem {
  data_category: string;
  policy_days: number;
  compliant: boolean;
}

export interface PiplComplianceData {
  compliance_status: string;
  data_handling_rules: DataHandlingRules;
  chinese_user_consent_log: ConsentLogEntry[];
  data_protection_officer: DpoInfo;
  cross_border_transfer_applications: CrossBorderApplication[];
  data_retention_compliance: RetentionItem[];
}

export function usePiplCompliance() {
  const [data, setData] = useState<PiplComplianceData | null>(null);
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
        compliance_status: "Compliant",
        data_handling_rules: { consent_required: true, data_minimization: true, purpose_limitation: true, cross_border_assessment: true },
        chinese_user_consent_log: [
          { user: "zhang.wei@ggid.cn", purpose: "Account creation", timestamp: "2d ago", withdrawn: false },
          { user: "li.ming@ggid.cn", purpose: "Marketing communications", timestamp: "5d ago", withdrawn: false },
          { user: "wang.fang@ggid.cn", purpose: "Data export", timestamp: "1w ago", withdrawn: true },
        ],
        data_protection_officer: { name: "Chen Yu", email: "dpo@ggid.cn" },
        cross_border_transfer_applications: [
          { applicant: "Analytics Team", data_type: "User behavior data", recipient_country: "United States", status: "approved", assessment_result: "Passed security assessment" },
          { applicant: "Support Team", data_type: "Support tickets", recipient_country: "Singapore", status: "pending", assessment_result: "Under review" },
          { applicant: "Marketing", data_type: "Email addresses", recipient_country: "Ireland", status: "pending", assessment_result: "Awaiting documentation" },
        ],
        data_retention_compliance: [
          { data_category: "Identity documents", policy_days: 180, compliant: true },
          { data_category: "Biometric data", policy_days: 90, compliant: true },
          { data_category: "Transaction records", policy_days: 1825, compliant: true },
          { data_category: "Marketing data", policy_days: 365, compliant: false },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
