import { useState, useCallback, useEffect } from "react";

export interface DecisionResult {
  decision: "allow" | "deny" | "not_applicable";
  evaluation_time_ms: number;
  obligations: string[];
}

export interface MatchedRule {
  policy_name: string;
  condition_path: string;
  effect: "allow" | "deny" | "not_applicable";
}

export interface ResolutionStep {
  attribute: string;
  source: string;
  value: string;
  resolved: boolean;
}

export interface PolicyAbacEvaluatorData {
  decision_result: DecisionResult;
  matched_rules: MatchedRule[];
  attribute_resolution_trace: ResolutionStep[];
}

export function usePolicyAbacEvaluator() {
  const [data, setData] = useState<PolicyAbacEvaluatorData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        decision_result: {
          decision: "allow",
          evaluation_time_ms: 3,
          obligations: ["log_access", "require_mfa_for_write"],
        },
        matched_rules: [
          { policy_name: "Finance Data Access", condition_path: "resource.classification == \"confidential\" AND user.department == \"finance\"", effect: "allow" },
          { policy_name: "Business Hours Access", condition_path: "env.time == \"business_hours\" AND action == \"read\"", effect: "allow" },
          { policy_name: "Deny External Access", condition_path: "env.location != \"on_premise\" AND resource.classification == \"confidential\"", effect: "deny" },
        ],
        attribute_resolution_trace: [
          { attribute: "user.department", source: "LDAP directory", value: "finance", resolved: true },
          { attribute: "user.role", source: "JWT claim", value: "analyst", resolved: true },
          { attribute: "resource.type", source: "resource registry", value: "document", resolved: true },
          { attribute: "resource.classification", source: "metadata tag", value: "confidential", resolved: true },
          { attribute: "env.time", source: "server clock", value: "business_hours", resolved: true },
          { attribute: "env.location", source: "IP geo-lookup", value: "on_premise", resolved: true },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const evaluate = useCallback(async (_userAttrs: string, _resourceAttrs: string, _envAttrs: string, _action: string) => {
    console.log("Evaluating ABAC decision");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, evaluate };
}
