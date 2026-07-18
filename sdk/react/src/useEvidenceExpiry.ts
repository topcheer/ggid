import { useState, useCallback } from "react";

export interface EvidenceItem {
  id: string;
  control_id: string;
  framework: string;
  evidence_type: string;
  collected_at: string;
  expires_at: string;
  days_remaining: number;
  status: "valid" | "expiring" | "expired";
}

export function useEvidenceExpiry(baseUrl: string = "") {
  const [items, setItems] = useState<EvidenceItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchExpiry = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/evidence-expiry`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setItems(data.evidence || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const refreshItem = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/evidence-expiry/${id}/refresh`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setItems((prev: any) => prev.map((e) => e.id === id ? { ...e, status: "valid", days_remaining: 90 } : e));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { items, loading, error, fetchExpiry, refreshItem };
}
