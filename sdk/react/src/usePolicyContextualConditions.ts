import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ConditionCategory {
  name: string;
  available_attributes: string[];
}

export interface ConditionTemplate {
  name: string;
  categories: string[];
  condition_summary: string;
  usage_count: number;
}

export interface PolicyContextualConditionsData {
  condition_categories: ConditionCategory[];
  saved_condition_templates: ConditionTemplate[];
}

export function usePolicyContextualConditions() {
  const [data, setData] = useState<PolicyContextualConditionsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        condition_categories: [
          { name: "time", available_attributes: ["hour_of_day", "day_of_week", "is_holiday", "timezone"] },
          { name: "geo", available_attributes: ["country", "region", "city", "ip_range", "distance_from_usual"] },
          { name: "device", available_attributes: ["device_type", "os_version", "browser", "is_managed", "attestation"] },
          { name: "network", available_attributes: ["ip_address", "asn", "is_vpn", "is_tor", "cidr_block"] },
          { name: "risk", available_attributes: ["risk_score", "threat_level", "anomaly_count", "failed_attempts"] },
          { name: "behavioral", available_attributes: ["login_velocity", "access_pattern_deviation", "typing_pattern_match"] },
        ],
        saved_condition_templates: [
          { name: "Business Hours Only", categories: ["time"], condition_summary: "hour_of_day >= 8 AND hour_of_day <= 18 AND day_of_week in [Mon-Fri]", usage_count: 12 },
          { name: "Corporate Network", categories: ["network"], condition_summary: "cidr_block in [10.0.0.0/8, 172.16.0.0/12]", usage_count: 8 },
          { name: "Low Risk + Managed Device", categories: ["risk", "device"], condition_summary: "risk_score < 30 AND is_managed = true", usage_count: 5 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testEvaluation = useCallback(async () => {
    console.log("Testing condition evaluation");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testEvaluation };
}
