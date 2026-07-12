import { useState, useCallback, useEffect } from "react";

export interface FieldMapEntry {
  local_field: string;
  siem_field: string;
}

export interface SeverityMapEntry {
  our_severity: string;
  siem_severity: string;
}

export interface FormatConfig {
  destination: string;
  template: string;
  field_mapping: FieldMapEntry[];
  severity_mapping: SeverityMapEntry[];
  sample_output: string;
  validation_passed: boolean;
}

export interface SiemLogFormatsData {
  format_configs: FormatConfig[];
  template_library: string[];
}

export function useSiemLogFormats() {
  const [data, setData] = useState<SiemLogFormatsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        format_configs: [
          { destination: "Splunk Production", template: "CEF", field_mapping: [
            { local_field: "event_type", siem_field: "CEF.version" },
            { local_field: "severity", siem_field: "CEF.severity" },
            { local_field: "user_id", siem_field: "CEF.extension.suser" },
            { local_field: "ip_address", siem_field: "CEF.extension.src" },
            { local_field: "action", siem_field: "CEF.extension.act" },
          ], severity_mapping: [
            { our_severity: "critical", siem_severity: "10" },
            { our_severity: "high", siem_severity: "8" },
            { our_severity: "medium", siem_severity: "5" },
            { our_severity: "low", siem_severity: "3" },
          ], sample_output: "CEF:0|GGID|IAM|1.0|auth.login|User login|5|suser=alice src=10.0.0.1 act=login", validation_passed: true },
          { destination: "QRadar SIEM", template: "LEEF", field_mapping: [
            { local_field: "event_type", siem_field: "LEEF.header.event" },
            { local_field: "severity", siem_field: "LEEF.severity" },
            { local_field: "user_id", siem_field: "LEEF.username" },
          ], severity_mapping: [
            { our_severity: "critical", siem_severity: "100" },
            { our_severity: "high", siem_severity: "80" },
            { our_severity: "medium", siem_severity: "50" },
            { our_severity: "low", siem_severity: "20" },
          ], sample_output: "LEEF:1.0|GGID|IAM|1.0|auth.login|sev=50\tusername=alice\tsrc=10.0.0.1", validation_passed: true },
        ],
        template_library: ["CEF (Common Event Format)", "LEEF (Log Event Extended Format)", "JSON", "Syslog", "Kafka Connect", "Custom HTTP"],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
