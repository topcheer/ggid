import { useState, useCallback, useEffect } from "react";

export interface PipelineStage {
  name: string;
  throughput: number;
  latency_ms: number;
  error_rate: number;
  queue_depth: number;
}

export interface Bottleneck {
  stage: string;
  severity: string;
  description: string;
  recommendation: string;
}

export interface FailoverStatus {
  primary_healthy: boolean;
  standby_ready: boolean;
  last_failover: string;
}

export interface LastIncident {
  description: string;
  duration: string;
  resolved_at: string;
}

export interface AuditPipelineHealthData {
  pipeline_stages: PipelineStage[];
  bottlenecks: Bottleneck[];
  failover_status: FailoverStatus;
  last_incident: LastIncident;
}

export function useAuditPipelineHealth() {
  const [data, setData] = useState<AuditPipelineHealthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        pipeline_stages: [
          { name: "Ingest", throughput: 5000, latency_ms: 2.1, error_rate: 0.05, queue_depth: 120 },
          { name: "Validate", throughput: 4800, latency_ms: 5.3, error_rate: 0.12, queue_depth: 340 },
          { name: "Enrich", throughput: 4500, latency_ms: 12.5, error_rate: 0.08, queue_depth: 890 },
          { name: "Store", throughput: 4300, latency_ms: 8.2, error_rate: 0.03, queue_depth: 450 },
          { name: "Index", throughput: 4100, latency_ms: 15.7, error_rate: 0.15, queue_depth: 1200 },
          { name: "Query", throughput: 4000, latency_ms: 3.5, error_rate: 0.02, queue_depth: 50 },
        ],
        bottlenecks: [
          { stage: "Index", severity: "warning", description: "Queue depth exceeds 1000 threshold", recommendation: "Scale index workers from 4 to 6" },
          { stage: "Enrich", severity: "info", description: "Latency trending upward (+15% vs yesterday)", recommendation: "Monitor correlation rules complexity" },
        ],
        failover_status: { primary_healthy: true, standby_ready: true, last_failover: "12d ago" },
        last_incident: { description: "Index stage degraded - queue backlog after bulk import", duration: "23m", resolved_at: "5d ago" },
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
