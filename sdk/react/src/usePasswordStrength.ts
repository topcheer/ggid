import { useState, useCallback } from "react";

export interface PasswordStrengthData {
  total_users: number;
  distribution: { weak: number; fair: number; good: number; strong: number };
  policy_compliance_pct: number;
  avg_entropy_bits: number;
  min_entropy: number;
  max_entropy: number;
  weak_passwords: { user_id: string; username: string; entropy_bits: number; last_changed: string }[];
}

export function usePasswordStrength(baseUrl: string = "") {
  const [data, setData] = useState<PasswordStrengthData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStrength = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/password-strength`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: PasswordStrengthData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchStrength };
}
