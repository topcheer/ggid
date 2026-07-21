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

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

/** Get auth token safely (SSR-safe). */
function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("ggid_access_token");
}

/** Build auth headers. */
function authHeaders(): Record<string, string> {
  const token = getToken();
  return {
    "Content-Type": "application/json",
    "X-Tenant-ID": TENANT_ID,
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

/**
 * ITDR Dashboard — connected to real backend endpoints.
 *
 * Endpoints:
 * - GET /api/v1/audit/itdr/stats — dashboard stats + auto_response_enabled
 * - GET /api/v1/audit/itdr/detections — threat detection list
 * - GET /api/v1/audit/itdr/rules — detection rules
 */
export function useITDRDashboard() {
  const [data, setData] = useState<ITDRDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isDemoData, setIsDemoData] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const headers = authHeaders();

      const [statsRes, detectionsRes, rulesRes] = await Promise.all([
        fetch(`/api/v1/audit/itdr/stats`, { headers }).catch(() => null),
        fetch(`/api/v1/audit/itdr/detections`, { headers }).catch(() => null),
        fetch(`/api/v1/audit/itdr/rules`, { headers }).catch(() => null),
      ]);

      // If all fail, show empty state (not fake data)
      if (!statsRes?.ok && !detectionsRes?.ok) {
        setData({
          threat_detections: [],
          detection_rules: [],
          response_playbooks: [],
          auto_response_enabled: false,
        });
        setIsDemoData(true);
        return;
      }

      const stats = statsRes?.ok ? await statsRes.json().catch(() => ({})) : {};
      const detections = detectionsRes?.ok ? await detectionsRes.json().catch(() => ({ detections: [] })) : { detections: [] };
      const rules = rulesRes?.ok ? await rulesRes.json().catch(() => ({ rules: [] })) : { rules: [] };

      setData({
        threat_detections: detections.detections || detections.items || [],
        detection_rules: rules.rules || rules.items || [],
        response_playbooks: stats.playbooks || [],
        // Backend must explicitly set this — never default to true
        auto_response_enabled: stats.auto_response_enabled === true,
      });
      setIsDemoData(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load ITDR data");
      setData({
        threat_detections: [],
        detection_rules: [],
        response_playbooks: [],
        auto_response_enabled: false,
      });
      setIsDemoData(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
