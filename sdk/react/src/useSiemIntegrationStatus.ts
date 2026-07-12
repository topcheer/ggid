import { useState, useCallback, useEffect } from "react";

export interface SiemDestination {
  id: string;
  name: string;
  type: "Splunk" | "QRadar" | "Elasticsearch" | "Custom";
  status: "connected" | "disconnected" | "error";
  throughput: number;
  queue_depth: number;
  retry_failures: number;
  last_sync: string;
}

export interface SiemIntegrationStatusData {
  total_throughput: number;
  total_queue_depth: number;
  total_retry_failures: number;
  destinations: SiemDestination[];
}

export function useSiemIntegrationStatus() {
  const [data, setData] = useState<SiemIntegrationStatusData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        total_throughput: 45200,
        total_queue_depth: 1200,
        total_retry_failures: 14,
        destinations: [
          {
            id: "dest-1",
            name: "Splunk Enterprise",
            type: "Splunk",
            status: "connected",
            throughput: 28000,
            queue_depth: 300,
            retry_failures: 2,
            last_sync: "30s ago",
          },
          {
            id: "dest-2",
            name: "IBM QRadar",
            type: "QRadar",
            status: "connected",
            throughput: 12000,
            queue_depth: 500,
            retry_failures: 5,
            last_sync: "1m ago",
          },
          {
            id: "dest-3",
            name: "Elasticsearch SIEM",
            type: "Elasticsearch",
            status: "error",
            throughput: 5200,
            queue_depth: 400,
            retry_failures: 7,
            last_sync: "15m ago",
          },
          {
            id: "dest-4",
            name: "Custom Webhook",
            type: "Custom",
            status: "connected",
            throughput: 0,
            queue_depth: 0,
            retry_failures: 0,
            last_sync: "5m ago",
          },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testConnection = useCallback(async (destId: string) => {
    console.log("Testing connection for destination:", destId);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testConnection };
}
