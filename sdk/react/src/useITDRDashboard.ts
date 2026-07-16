import { useState, useCallback, useEffect } from "react";

export interface ThreatDetection {
  id: string;
  type: string;
  severity: string;
  source: string;
  timestamp: string;
  affected_users: number;
  status: string;
  mitre_techniques: string[];
}

export interface DetectionRule {
  rule_name: string;
  technique: string;
  enabled: boolean;
  last_triggered: string;
}

export interface ResponsePlaybook {
  threat_type: string;
  steps_count: number;
  estimated_time: string;
  auto_execute: boolean;
}

export interface ITDRDashboardData {
  threat_detections: ThreatDetection[];
  detection_rules: DetectionRule[];
  response_playbooks: ResponsePlaybook[];
  auto_response_enabled: boolean;
}

/**
 * DEMO DATA — No backend ITDR API exists yet.
 * All data shown is fictional. Do NOT use for operational decisions.
 *
 * TODO: Replace with real fetch when backend implements:
 *   GET /api/v1/security/itdr/dashboard
 *
 * Safety: auto_response_enabled is hardcoded to false (never simulate
 * active auto-response — SOC operators could misinterpret it as real).
 */
export function useITDRDashboard() {
  const [data, setData] = useState<ITDRDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      const res = await fetch("/api/v1/security/itdr/dashboard", {
        headers: { "Content-Type": "application/json" },
      }).catch(() => null);

      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }

      // Fallback to demo data — clearly marked
      // auto_response_enabled MUST be false in demo mode
      await new Promise((r) => setTimeout(r, 400));
      setData({
        threat_detections: [],
        detection_rules: [],
        response_playbooks: [],
        auto_response_enabled: false,
      });
      setIsDemoData(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
