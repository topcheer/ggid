import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface MonitoredAgent {
  agent_id: string;
  agent_name: string;
  normal_baseline: string;
  current_behavior: string;
  deviation_score: number;
}

export interface AnomalyAlert {
  agent_name: string;
  type: string;
  description: string;
  timestamp: string;
}

export interface AgentBehavioralMonitorData {
  monitored_agents: MonitoredAgent[];
  anomaly_alerts: AnomalyAlert[];
  auto_suspend_threshold: number;
}

export function useAgentBehavioralMonitor() {
  const [data, setData] = useState<AgentBehavioralMonitorData | null>(null);
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
        monitored_agents: [
          { agent_id: "agent-001", agent_name: "CI/CD Bot", normal_baseline: "45 req/min", current_behavior: "42 req/min", deviation_score: 0.05 },
          { agent_id: "agent-002", agent_name: "Monitoring Agent", normal_baseline: "180 req/min", current_behavior: "420 req/min", deviation_score: 0.82 },
          { agent_id: "agent-003", agent_name: "Data Pipeline", normal_baseline: "12 req/min", current_behavior: "0 req/min", deviation_score: 0.15 },
          { agent_id: "agent-004", agent_name: "Security Scanner", normal_baseline: "8 req/min", current_behavior: "15 req/min", deviation_score: 0.47 },
        ],
        anomaly_alerts: [
          { agent_name: "Monitoring Agent", type: "excessive_requests", description: "Request rate 2.3x higher than baseline (420 vs 180 req/min)", timestamp: "10m ago" },
          { agent_name: "Monitoring Agent", type: "unusual_api_pattern", description: "Accessing /api/v1/admin/config outside declared scope", timestamp: "20m ago" },
          { agent_name: "Security Scanner", type: "off_hours_access", description: "API calls at 03:47 UTC, outside normal activity window", timestamp: "2h ago" },
        ],
        auto_suspend_threshold: 0.8,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
