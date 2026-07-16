import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RetentionPolicy {
  data_type: string;
  retention_days: number;
  action: string;
  legal_basis: string;
}

export interface StorageAge {
  age_range: string;
  size_gb: number;
  pct: number;
}

export interface UpcomingPurge {
  date: string;
  data_type: string;
  affected_records: number;
}

export interface ComplianceEntry {
  framework: string;
  compliant: boolean;
}

export interface PurgeHistoryEntry {
  date: string;
  data_type: string;
  records_purged: number;
}

export interface DataRetentionDashboardData {
  retention_policies: RetentionPolicy[];
  storage_usage_by_age: StorageAge[];
  upcoming_purges: UpcomingPurge[];
  compliance_status: ComplianceEntry[];
  purge_history: PurgeHistoryEntry[];
}

export function useDataRetentionDashboard() {
  const [data, setData] = useState<DataRetentionDashboardData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
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
        retention_policies: [
          { data_type: "Audit Logs", retention_days: 2555, action: "archive", legal_basis: "SOC2 Type II" },
          { data_type: "User Sessions", retention_days: 90, action: "delete", legal_basis: "GDPR Art. 5(1)(e)" },
          { data_type: "PII Data", retention_days: 365, action: "anonymize", legal_basis: "GDPR Art. 17" },
          { data_type: "Security Events", retention_days: 1095, action: "archive", legal_basis: "PCI-DSS Req 10" },
        ],
        storage_usage_by_age: [
          { age_range: "0-30d", size_gb: 45, pct: 35 },
          { age_range: "31-90d", size_gb: 32, pct: 25 },
          { age_range: "91-365d", size_gb: 28, pct: 22 },
          { age_range: "1-3y", size_gb: 15, pct: 12 },
          { age_range: "3y+", size_gb: 8, pct: 6 },
        ],
        upcoming_purges: [
          { date: "2024-03-20", data_type: "Expired Sessions", affected_records: 125000 },
          { date: "2024-04-01", data_type: "Anonymized PII", affected_records: 3400 },
        ],
        compliance_status: [
          { framework: "GDPR", compliant: true },
          { framework: "CCPA", compliant: true },
          { framework: "SOC2", compliant: true },
          { framework: "PCI-DSS", compliant: false },
        ],
        purge_history: [
          { date: "2024-03-01", data_type: "Sessions", records_purged: 89000 },
          { date: "2024-02-15", data_type: "Audit Logs", records_purged: 450000 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
