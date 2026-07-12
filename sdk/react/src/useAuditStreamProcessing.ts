import { useState, useCallback, useEffect } from "react";

export interface StreamHealth {
  input_rate: number;
  processing_rate: number;
  output_rate: number;
  lag_seconds: number;
}

export interface LagPoint {
  timestamp: string;
  lag_seconds: number;
}

export interface RetryPolicy {
  max_retries: number;
  backoff_strategy: string;
  initial_delay_ms: number;
  max_delay_ms: number;
}

export interface ScalingConfig {
  auto_scale_threshold: number;
  min_consumers: number;
  max_consumers: number;
  current_consumers: number;
}

export interface AuditStreamProcessingData {
  stream_health: StreamHealth;
  consumer_lag_history: LagPoint[];
  dead_letter_queue_count: number;
  retry_policy: RetryPolicy;
  backpressure_status: string;
  scaling_config: ScalingConfig;
}

export function useAuditStreamProcessing() {
  const [data, setData] = useState<AuditStreamProcessingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      const lagHistory: LagPoint[] = Array.from({ length: 30 }, (_, i) => ({
        timestamp: `-${(30 - i) * 2}s`,
        lag_seconds: Math.max(1, Math.round(3 + Math.sin(i / 3) * 4 + Math.random() * 2)),
      }));
      setData({
        stream_health: { input_rate: 4200, processing_rate: 4180, output_rate: 4150, lag_seconds: 4 },
        consumer_lag_history: lagHistory,
        dead_letter_queue_count: 3,
        retry_policy: { max_retries: 5, backoff_strategy: "exponential", initial_delay_ms: 100, max_delay_ms: 10000 },
        backpressure_status: "normal",
        scaling_config: { auto_scale_threshold: 5000, min_consumers: 2, max_consumers: 10, current_consumers: 4 },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData };
}
