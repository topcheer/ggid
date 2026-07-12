import { useState, useCallback, useEffect } from "react";

export interface GroupRule {
  group_name: string;
  rule_expression: string;
  membership_type: "dynamic" | "static" | "hybrid";
  member_count: number;
  preview_members: PreviewMember[];
}

export interface PreviewMember {
  username: string;
  matched_attribute: string;
}

export interface IdentityDynamicGroupingData {
  group_rules: GroupRule[];
  evaluation_frequency: string;
  conflict_resolution: string;
}

export function useIdentityDynamicGrouping() {
  const [data, setData] = useState<IdentityDynamicGroupingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        group_rules: [
          { group_name: "Engineering Team", rule_expression: "department = 'Engineering'", membership_type: "dynamic", member_count: 24, preview_members: [
            { username: "alice.chen", matched_attribute: "Engineering" },
            { username: "bob.martinez", matched_attribute: "Engineering" },
            { username: "carol.jones", matched_attribute: "Engineering" },
          ]},
          { group_name: "US Office", rule_expression: "location in ['US-NYC', 'US-SF', 'US-LA']", membership_type: "dynamic", member_count: 45, preview_members: [
            { username: "dave.wilson", matched_attribute: "US-NYC" },
            { username: "eve.brown", matched_attribute: "US-SF" },
          ]},
          { group_name: "Managers + Direct", rule_expression: "title contains 'Manager' OR title contains 'Director'", membership_type: "hybrid", member_count: 12, preview_members: [
            { username: "frank.lee", matched_attribute: "Manager" },
          ]},
        ],
        evaluation_frequency: "real-time",
        conflict_resolution: "priority",
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const evaluatePreview = useCallback((_group: string) => {
    console.log("Evaluating group:", _group);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, evaluatePreview };
}
