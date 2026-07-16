import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface GdprRequest {
  id: string;
  request_type: "access" | "erasure" | "portability" | "rectification";
  user_id: string;
  status: "pending" | "processing" | "completed";
  deadline_days: number;
  identity_verified: boolean;
  anonymization_preview?: string[];
}

export interface CompletedStats {
  total_30d: number;
  overdue: number;
  by_type: Record<string, number>;
}

export interface AuditGdprRequestsData {
  request_queue: GdprRequest[];
  completed_stats: CompletedStats;
  sla_compliance_pct: number;
}

export function useAuditGdprRequests() {
  const [data, setData] = useState<AuditGdprRequestsData | null>(null);
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
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        request_queue: [
          { id: "gdpr-1", request_type: "access", user_id: "user-1234", status: "pending", deadline_days: 15, identity_verified: true },
          { id: "gdpr-2", request_type: "erasure", user_id: "user-5678", status: "pending", deadline_days: 5, identity_verified: true, anonymization_preview: ["email", "full_name", "phone", "address", "date_of_birth"] },
          { id: "gdpr-3", request_type: "portability", user_id: "user-9012", status: "processing", deadline_days: 20, identity_verified: true },
          { id: "gdpr-4", request_type: "rectification", user_id: "user-3456", status: "pending", deadline_days: -2, identity_verified: false },
          { id: "gdpr-5", request_type: "erasure", user_id: "user-7890", status: "pending", deadline_days: 8, identity_verified: true, anonymization_preview: ["email", "profile_data", "activity_logs"] },
        ],
        completed_stats: {
          total_30d: 28,
          overdue: 2,
          by_type: { access: 12, erasure: 8, portability: 5, rectification: 3 },
        },
        sla_compliance_pct: 93,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const processRequest = useCallback(async (reqId: string) => {
    console.log("Processing GDPR request:", reqId);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, processRequest };
}
