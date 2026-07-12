import { useState, useCallback, useEffect } from "react";

export interface FieldMapping {
  local_field: string;
  siem_field: string;
}

export interface SiemDestinationMapping {
  destination: string;
  field_mappings: FieldMapping[];
}

export interface SeverityMap {
  our_severity: string;
  siem_severity: string;
}

export interface CustomField {
  key: string;
  value: string;
}

export interface AuditSiemMappingData {
  per_destination: SiemDestinationMapping[];
  event_type_filter: string[];
  severity_mapping: SeverityMap[];
  custom_fields: CustomField[];
  throughput_estimate: number;
}

export function useAuditSiemMapping() {
  const [data, setData] = useState<AuditSiemMappingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        per_destination: [
          {
            destination: "Splunk",
            field_mappings: [
              { local_field: "event_id", siem_field: "_key" },
              { local_field: "timestamp", siem_field: "_time" },
              { local_field: "user_id", siem_field: "user" },
              { local_field: "action", siem_field: "event_type" },
              { local_field: "resource", siem_field: "object" },
              { local_field: "source_ip", siem_field: "src_ip" },
              { local_field: "tenant_id", siem_field: "tenant" },
              { local_field: "outcome", siem_field: "result" },
            ],
          },
        ],
        event_type_filter: [
          "auth.login.success", "auth.login.failed", "auth.logout",
          "role.change", "permission.grant", "permission.revoke",
          "policy.create", "policy.update", "policy.delete",
          "user.create", "user.delete", "user.suspend",
          "api.key.create", "api.key.revoke",
          "session.revoke", "token.revoke",
        ],
        severity_mapping: [
          { our_severity: "critical", siem_severity: "SEV1" },
          { our_severity: "high", siem_severity: "SEV2" },
          { our_severity: "medium", siem_severity: "SEV3" },
          { our_severity: "low", siem_severity: "SEV4" },
          { our_severity: "info", siem_severity: "INFO" },
        ],
        custom_fields: [
          { key: "product", value: "ggid" },
          { key: "env", value: "prod" },
          { key: "data_center", value: "us-west-2" },
        ],
        throughput_estimate: 4200,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testMapping = useCallback(async () => {
    console.log("Testing SIEM mapping");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testMapping };
}
