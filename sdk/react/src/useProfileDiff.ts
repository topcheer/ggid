import { useState, useCallback } from "react";

export interface DiffEntry {
  field: string;
  old_value: string;
  new_value: string;
  changed_by: string;
  changed_at: string;
}

export interface DiffResult {
  user_id: string;
  username: string;
  version_a: string;
  version_b: string;
  diffs: DiffEntry[];
}

export function useProfileDiff(baseUrl: string = "") {
  const [result, setResult] = useState<DiffResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const diff = useCallback(async (userId: string, versionA: string, versionB: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/profile-diff?user_id=${encodeURIComponent(userId)}&a=${encodeURIComponent(versionA)}&b=${encodeURIComponent(versionB)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setResult(await res.json());
    } catch (e: any) { setError(e.message); setResult(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { result, loading, error, diff };
}
