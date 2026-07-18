import { useState, useCallback } from "react";

export interface MFAFactor {
  id: string;
  type: "totp" | "webauthn" | "sms" | "backup";
  label: string;
  enabled: boolean;
  enrolled_at: string;
  last_used: string | null;
}

export function useMFAFactors(baseUrl: string = "") {
  const [factors, setFactors] = useState<MFAFactor[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchFactors = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/mfa-factors?user=${encodeURIComponent(user)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setFactors(data.factors || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const deleteFactor = useCallback(async (factorId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/mfa-factors/${factorId}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setFactors((prev: any) => prev.filter((f) => f.id !== factorId));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { factors, loading, error, fetchFactors, deleteFactor };
}
