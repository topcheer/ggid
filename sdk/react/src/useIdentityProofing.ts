import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ProofingStep {
  step: "document_upload" | "liveness_check" | "kba" | "manual_review";
  status: "pending" | "in_progress" | "completed" | "failed";
  description: string;
  confidence?: number;
}

export interface RecentVerification {
  user_name: string;
  document_type: string;
  status: "approved" | "rejected" | "pending";
  confidence: number;
  timestamp: string;
}

export interface IdentityProofingData {
  completion_rate: number;
  confidence_threshold: number;
  in_progress_count: number;
  proofing_steps: ProofingStep[];
  recent_verifications: RecentVerification[];
}

export function useIdentityProofing() {
  const [data, setData] = useState<IdentityProofingData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        completion_rate: 87,
        confidence_threshold: 0.85,
        in_progress_count: 12,
        proofing_steps: [
          { step: "document_upload", status: "completed", description: "User uploaded passport photo", confidence: 0.98 },
          { step: "liveness_check", status: "completed", description: "Selfie liveness verification passed", confidence: 0.94 },
          { step: "kba", status: "completed", description: "Knowledge-based authentication (3/3 correct)", confidence: 0.91 },
          { step: "manual_review", status: "in_progress", description: "Analyst reviewing document edges", confidence: undefined },
        ],
        recent_verifications: [
          { user_name: "Alice Chen", document_type: "Passport", status: "approved", confidence: 0.96, timestamp: "10m ago" },
          { user_name: "Bob Martinez", document_type: "Driver License", status: "approved", confidence: 0.92, timestamp: "1h ago" },
          { user_name: "Carol Jones", document_type: "National ID", status: "rejected", confidence: 0.45, timestamp: "2h ago" },
          { user_name: "Dave Wilson", document_type: "Passport", status: "pending", confidence: 0.0, timestamp: "3h ago" },
          { user_name: "Eve Brown", document_type: "Residence Permit", status: "approved", confidence: 0.89, timestamp: "5h ago" },
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
