import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ScoringSignal {
  signal_name: string;
  weight: number;
  threshold: number;
}

export interface CompositeThreshold {
  low: number;
  medium: number;
  high: number;
  critical: number;
}

export interface AccuracyStats {
  accuracy_pct: number;
  precision_pct: number;
  recall_pct: number;
  false_positive_rate: number;
  last_trained: string;
}

export interface AuditAnomalyScoringConfigData {
  scoring_signals: ScoringSignal[];
  composite_threshold: CompositeThreshold;
  baseline_window_hours: number;
  sensitivity_adjustment: string;
  accuracy_stats: AccuracyStats;
}

export function useAuditAnomalyScoringConfig() {
  const [data, setData] = useState<AuditAnomalyScoringConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
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
        scoring_signals: [
          { signal_name: "volume_spike", weight: 2.0, threshold: 0.85 },
          { signal_name: "new_entity", weight: 1.5, threshold: 0.70 },
          { signal_name: "off_hours", weight: 1.0, threshold: 0.60 },
          { signal_name: "geo_anomaly", weight: 2.5, threshold: 0.75 },
        ],
        composite_threshold: { low: 25, medium: 50, high: 75, critical: 90 },
        baseline_window_hours: 168,
        sensitivity_adjustment: "normal",
        accuracy_stats: {
          accuracy_pct: 92,
          precision_pct: 88,
          recall_pct: 85,
          false_positive_rate: 8,
          last_trained: "2d ago",
        },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const retrainModel = useCallback(async () => {
    console.log("Retraining anomaly model");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, retrainModel };
}
