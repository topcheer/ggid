import { useState, useCallback, useEffect } from "react";

export interface PIISource {
  table: string;
  column: string;
  pii_type: string;
  sample_masked: string;
  confidence: number;
  encrypted: boolean;
}

export interface UnencryptedAlert {
  location: string;
  pii_type: string;
}

export interface DatabaseBreakdown {
  database: string;
  pii_columns: number;
  unencrypted: number;
}

export interface PIIDiscoveryData {
  data_sources: PIISource[];
  unencrypted_pii_alerts: UnencryptedAlert[];
  coverage_pct: number;
  per_database_breakdown: DatabaseBreakdown[];
}

export function usePIIDiscovery() {
  const [data, setData] = useState<PIIDiscoveryData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        data_sources: [
          { table: "users", column: "email", pii_type: "email", sample_masked: "j***@e*****.com", confidence: 99, encrypted: true },
          { table: "users", column: "phone", pii_type: "phone", sample_masked: "+1-555-***-1234", confidence: 95, encrypted: true },
          { table: "users", column: "full_name", pii_type: "name", sample_masked: "J*** D**", confidence: 92, encrypted: false },
          { table: "user_profiles", column: "ssn", pii_type: "SSN", sample_masked: "***-**-5678", confidence: 88, encrypted: true },
          { table: "user_profiles", column: "address", pii_type: "address", sample_masked: "123 M*** St", confidence: 76, encrypted: false },
          { table: "audit_logs", column: "ip_address", pii_type: "IP", sample_masked: "192.168.*.*", confidence: 65, encrypted: false },
        ],
        unencrypted_pii_alerts: [
          { location: "users.full_name", pii_type: "name" },
          { location: "user_profiles.address", pii_type: "address" },
        ],
        coverage_pct: 87,
        per_database_breakdown: [
          { database: "identity_db", pii_columns: 12, unencrypted: 2 },
          { database: "audit_db", pii_columns: 5, unencrypted: 1 },
          { database: "analytics_db", pii_columns: 3, unencrypted: 0 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
