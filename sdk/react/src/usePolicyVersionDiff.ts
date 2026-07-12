import { useState, useCallback } from "react";

export interface VersionDiff {
  version_a: string;
  version_b: string;
  field_changes: { field: string; change_type: "added" | "removed" | "modified"; old_value: string; new_value: string }[];
  impact_summary: { affected_users: number; affected_resources: number; rules_changed: number };
  breaking_changes: string[];
}

export function usePolicyVersionDiff(baseUrl: string = "") {
  const [data, setData] = useState<VersionDiff | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const diff = useCallback(async (policyId: string, versionA: string, versionB: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/version-diff?id=${encodeURIComponent(policyId)}&a=${encodeURIComponent(versionA)}&b=${encodeURIComponent(versionB)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, diff };
}
