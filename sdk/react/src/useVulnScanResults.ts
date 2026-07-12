import { useState, useCallback, useEffect } from "react";

export interface ScanRun {
  date: string;
  scanner: string;
  scope: string;
  total: number;
  critical_high: number;
}

export interface VulnFinding {
  cve: string;
  cvss: number;
  description: string;
  affected_component: string;
  fix_available: boolean;
  status: string;
  severity: string;
}

export interface VulnScanResultsData {
  scan_runs: ScanRun[];
  findings: VulnFinding[];
}

export function useVulnScanResults() {
  const [data, setData] = useState<VulnScanResultsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        scan_runs: [
          { date: "2024-03-10", scanner: "Trivy", scope: "Container images", total: 18, critical_high: 3 },
          { date: "2024-03-08", scanner: "Snyk", scope: "Go dependencies", total: 12, critical_high: 1 },
          { date: "2024-03-05", scanner: "GoSec", scope: "Source code", total: 8, critical_high: 0 },
          { date: "2024-03-01", scanner: "Nuclei", scope: "Web endpoints", total: 5, critical_high: 2 },
        ],
        findings: [
          { cve: "CVE-2024-1234", cvss: 9.8, description: "RCE in outdated openssl library", affected_component: "openssl:1.1.1k", fix_available: true, status: "open", severity: "Critical" },
          { cve: "CVE-2024-5678", cvss: 8.2, description: "SQL injection in user query", affected_component: "identity-service", fix_available: true, status: "fixed", severity: "High" },
          { cve: "CVE-2024-9012", cvss: 6.5, description: "Information disclosure in error responses", affected_component: "gateway", fix_available: false, status: "open", severity: "Medium" },
          { cve: "CVE-2024-3456", cvss: 4.3, description: "Missing security header X-Frame-Options", affected_component: "console", fix_available: true, status: "fixed", severity: "Low" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
