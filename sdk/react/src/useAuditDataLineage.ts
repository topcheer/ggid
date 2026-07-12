import { useState, useCallback, useEffect } from "react";

export interface FlowNode {
  id: string;
  name: string;
  type: string;
}

export interface DatasetLineage {
  dataset_name: string;
  source_system: string;
  transformations: string[];
  downstream_consumers: string[];
  pii_classification: string;
  retention_path: string;
  deletion_propagation: string;
}

export interface AuditDataLineageData {
  flow_nodes: FlowNode[];
  datasets: DatasetLineage[];
}

export function useAuditDataLineage() {
  const [data, setData] = useState<AuditDataLineageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        flow_nodes: [
          { id: "n1", name: "PostgreSQL", type: "source" },
          { id: "n2", name: "ETL Pipeline", type: "processor" },
          { id: "n3", name: "Data Warehouse", type: "processor" },
          { id: "n4", name: "BI Dashboard", type: "destination" },
        ],
        datasets: [
          {
            dataset_name: "user_profiles",
            source_system: "PostgreSQL",
            transformations: ["PII masking", "field mapping", "aggregation"],
            downstream_consumers: ["BI Dashboard", "Analytics API", "ML Pipeline"],
            pii_classification: "high",
            retention_path: "90 days -> archive",
            deletion_propagation: "cascade",
          },
          {
            dataset_name: "audit_logs",
            source_system: "NATS JetStream",
            transformations: ["hash chaining", "compression"],
            downstream_consumers: ["SIEM Splunk", "Archive S3"],
            pii_classification: "medium",
            retention_path: "7 years",
            deletion_propagation: "manual_only",
          },
          {
            dataset_name: "session_metrics",
            source_system: "Redis",
            transformations: ["anonymization", "aggregation"],
            downstream_consumers: ["Grafana", "Prometheus"],
            pii_classification: "none",
            retention_path: "30 days",
            deletion_propagation: "automatic",
          },
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
