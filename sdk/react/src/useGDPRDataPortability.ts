import { useState, useCallback, useEffect } from "react";

export interface ExportRequest {
  user: string;
  requested_at: string;
  status: string;
  format: string;
  scope: string[];
  download_link: string;
}

export interface GDPRDataPortabilityData {
  export_requests: ExportRequest[];
  auto_expiry_days: number;
}

export function useGDPRDataPortability() {
  const [data, setData] = useState<GDPRDataPortabilityData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        export_requests: [
          { user: "alice@corp.com", requested_at: "2024-03-10 10:30", status: "ready", format: "JSON", scope: ["profile", "activity", "consents"], download_link: "/exports/alice_20240310.json" },
          { user: "bob@corp.com", requested_at: "2024-03-12 14:00", status: "processing", format: "CSV", scope: ["profile", "sessions"], download_link: "" },
          { user: "charlie@corp.com", requested_at: "2024-03-08 09:15", status: "expired", format: "JSON", scope: ["profile", "activity", "consents", "sessions", "audit_events"], download_link: "" },
          { user: "diana@corp.com", requested_at: "2024-03-13 16:45", status: "queued", format: "JSON", scope: ["profile"], download_link: "" },
        ],
        auto_expiry_days: 7,
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const generateExport = useCallback(async (user: string, scopes: string[]) => {
    console.log("Generating export for", user, "scopes:", scopes);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, generateExport };
}
