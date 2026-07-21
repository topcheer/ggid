import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ScanResult {
  client_id: string;
  secret_in_source: boolean;
  last_rotated_days: number;
  exposure_risk: "low" | "medium" | "high" | "critical";
}

export interface SecretFinding {
  file: string;
  line: number;
  preview_masked: string;
  severity: "critical" | "warning";
}

export interface OAuthClientSecretScannerData {
  codebase_scan_enabled: boolean;
  git_history_scan_enabled: boolean;
  scan_frequency: string;
  scan_results: ScanResult[];
  secrets_found: SecretFinding[];
}

export function useOAuthClientSecretScanner() {
  const [data, setData] = useState<OAuthClientSecretScannerData | null>(null);
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
        codebase_scan_enabled: false,
        git_history_scan_enabled: false,
        scan_frequency: "daily",
        scan_results: [
          { client_id: "client-web-001", secret_in_source: false, last_rotated_days: 12, exposure_risk: "low" },
          { client_id: "client-mobile-002", secret_in_source: true, last_rotated_days: 45, exposure_risk: "critical" },
          { client_id: "client-api-003", secret_in_source: false, last_rotated_days: 5, exposure_risk: "low" },
          { client_id: "client-legacy-004", secret_in_source: true, last_rotated_days: 180, exposure_risk: "high" },
          { client_id: "client-spa-005", secret_in_source: false, last_rotated_days: 30, exposure_risk: "medium" },
        ],
        secrets_found: [
          { file: "config/production.yml", line: 42, preview_masked: "client_secret: ****Xk9m", severity: "critical" },
          { file: "src/auth/oauth.ts", line: 18, preview_masked: "secret: '****aB3n'", severity: "critical" },
          { file: "scripts/deploy.sh", line: 7, preview_masked: "export SECRET=****", severity: "warning" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const autoRotateExposed = useCallback(async () => {
    console.log("Auto-rotating exposed secrets");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, autoRotateExposed };
}
