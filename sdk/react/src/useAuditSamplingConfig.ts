import { useState, useCallback, useEffect } from "react";

export interface SamplingStrategy {
  strategy: string;
  sample_size_pct: number;
  target_population: string;
  last_review: string | null;
}

export interface EventTypeRate {
  event_type: string;
  sampling_rate: number;
  volume: number;
}

export interface PopulationStats {
  total_events: number;
  sampled: number;
  unsampled: number;
}

export interface AuditSamplingConfigData {
  sampling_strategies: SamplingStrategy[];
  per_event_type_rate: EventTypeRate[];
  confidence_interval_target: number;
  last_sample_review: string;
  population_stats: PopulationStats;
}

export function useAuditSamplingConfig() {
  const [data, setData] = useState<AuditSamplingConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        sampling_strategies: [
          { strategy: "random", sample_size_pct: 10, target_population: "All events", last_review: "2d ago" },
          { strategy: "stratified", sample_size_pct: 25, target_population: "By event type", last_review: "1d ago" },
          { strategy: "risk_weighted", sample_size_pct: 50, target_population: "High-risk events", last_review: "3h ago" },
          { strategy: "systematic", sample_size_pct: 5, target_population: "Every Nth event", last_review: "5d ago" },
        ],
        per_event_type_rate: [
          { event_type: "auth.login", sampling_rate: 1.0, volume: 12000 },
          { event_type: "auth.failed_login", sampling_rate: 1.0, volume: 800 },
          { event_type: "api.request", sampling_rate: 0.05, volume: 500000 },
          { event_type: "policy.evaluated", sampling_rate: 0.1, volume: 45000 },
          { event_type: "data.access", sampling_rate: 0.5, volume: 30000 },
          { event_type: "admin.action", sampling_rate: 1.0, volume: 500 },
          { event_type: "config.change", sampling_rate: 1.0, volume: 200 },
        ],
        confidence_interval_target: 95,
        last_sample_review: "3h ago",
        population_stats: { total_events: 588500, sampled: 49350, unsampled: 539150 },
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
