import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface TagEntry {
  name: string;
  category: "category" | "env" | "compliance" | "custom";
  usage_count: number;
}

export interface TaggedPolicy {
  policy_id: string;
  policy_name: string;
  status: string;
  tags: string[];
}

export interface AutoTagRule {
  id: string;
  name: string;
  condition: string;
  applied_tags: string[];
  enabled: boolean;
}

export interface PolicyTaggingData {
  tag_taxonomy: TagEntry[];
  tagged_policies: TaggedPolicy[];
  auto_tag_rules: AutoTagRule[];
}

export function usePolicyTagging() {
  const [data, setData] = useState<PolicyTaggingData | null>(null);
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
        tag_taxonomy: [
          { name: "security", category: "category", usage_count: 28 },
          { name: "authentication", category: "category", usage_count: 22 },
          { name: "access-control", category: "category", usage_count: 18 },
          { name: "dev", category: "env", usage_count: 12 },
          { name: "staging", category: "env", usage_count: 8 },
          { name: "prod", category: "env", usage_count: 15 },
          { name: "SOC2", category: "compliance", usage_count: 9 },
          { name: "GDPR", category: "compliance", usage_count: 7 },
          { name: "HIPAA", category: "compliance", usage_count: 4 },
          { name: "legacy", category: "custom", usage_count: 3 },
        ],
        tagged_policies: [
          { policy_id: "pol-001", policy_name: "MFA Required for Admin API", status: "active", tags: ["security", "authentication", "prod", "SOC2"] },
          { policy_id: "pol-002", policy_name: "Session Timeout 15min", status: "active", tags: ["security", "prod"] },
          { policy_id: "pol-003", policy_name: "Dev Environment Access", status: "active", tags: ["access-control", "dev"] },
          { policy_id: "pol-004", policy_name: "GDPR Data Retention", status: "active", tags: ["compliance", "GDPR", "prod"] },
          { policy_id: "pol-005", policy_name: "HIPAA Audit Logging", status: "active", tags: ["compliance", "HIPAA", "prod"] },
          { policy_id: "pol-006", policy_name: "Legacy Auth Fallback", status: "deprecated", tags: ["authentication", "legacy"] },
          { policy_id: "pol-007", policy_name: "Staging IP Allowlist", status: "active", tags: ["access-control", "staging"] },
          { policy_id: "pol-008", policy_name: "SOC2 Access Review", status: "draft", tags: ["compliance", "SOC2"] },
        ],
        auto_tag_rules: [
          { id: "rule-1", name: "Tag prod policies", condition: "environment = prod", applied_tags: ["prod"], enabled: true },
          { id: "rule-2", name: "Tag MFA policies as security", condition: "action CONTAINS mfa", applied_tags: ["security", "authentication"], enabled: true },
          { id: "rule-3", name: "Tag compliance policies", condition: "framework IS NOT NULL", applied_tags: ["compliance"], enabled: true },
          { id: "rule-4", name: "Tag deprecated as legacy", condition: "status = deprecated", applied_tags: ["legacy"], enabled: false },
        ],
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

  return { data, loading, error, refresh: fetchData, isDemoData };
}
