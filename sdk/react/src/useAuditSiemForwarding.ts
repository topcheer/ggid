import { useState, useCallback, useEffect } from "react";

export interface SiemDestination {
  id: string;
  name: string;
  type: string;
  format: string;
  status: string;
  throughput_events_per_min: number;
  queue_depth: number;
  events_forwarded_24h: number;
  last_error: string;
  event_filter: string[];
}

export interface AuditSiemForwardingData {
  destinations: SiemDestination[];
}

export function useAuditSiemForwarding() {
  const [data, setData] = useState<AuditSiemForwardingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        destinations: [
          { id: "spl-1", name: "Splunk Production", type: "Splunk", format: "CEF", status: "connected", throughput_events_per_min: 2400, queue_depth: 12, events_forwarded_24h: 3456000, last_error: "", event_filter: ["auth.login", "auth.logout", "policy.decision", "admin.action", "security.alert"] },
          { id: "qr-1", name: "QRadar SIEM", type: "QRadar", format: "LEEF", status: "connected", throughput_events_per_min: 1800, queue_depth: 0, events_forwarded_24h: 2592000, last_error: "", event_filter: ["auth.*", "oauth.*", "policy.*", "audit.*"] },
          { id: "es-1", name: "Elastic Security", type: "Elasticsearch", format: "JSON", status: "error", throughput_events_per_min: 0, queue_depth: 8500, events_forwarded_24h: 980000, last_error: "Connection timeout to elasticsearch:9200", event_filter: ["security.*", "auth.*"] },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const testForward = useCallback((destId: string) => { console.log("Testing forward to", destId); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, testForward };
}
