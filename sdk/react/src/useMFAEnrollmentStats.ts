import { useState, useCallback } from "react";

export interface MFAStats {
  method_distribution: { method: string; count: number }[];
  enrollment_rate: number;
  avg_methods_per_user: number;
  pending_enrollments: { user_id: string; username: string; method: string; initiated_at: string }[];
}

export function useMFAEnrollmentStats(baseUrl: string = "") {
  const [data, setData] = useState<MFAStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/mfa-enrollment-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
