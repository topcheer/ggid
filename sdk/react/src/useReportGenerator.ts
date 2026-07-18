/**
 * GGID React SDK — useReportGenerator hook
 *
 * Compliance report generation and download.
 *
 * Usage:
 *   const { reports, generate, download, isLoading } = useReportGenerator();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type ReportFormat = 'pdf' | 'csv' | 'json';
export type ReportStatus = 'pending' | 'generating' | 'completed' | 'failed';

export interface GeneratedReport {
  id: string;
  framework: string;
  start_date: string;
  end_date: string;
  format: ReportFormat;
  status: ReportStatus;
  created_at: string;
  completed_at: string;
  file_size: number;
  download_url: string;
}

export interface GenerateReportInput {
  framework: string;
  start_date: string;
  end_date: string;
  format: ReportFormat;
}

export interface UseReportGeneratorResult {
  reports: GeneratedReport[];
  isLoading: boolean;
  error: string | null;
  fetchReports: () => Promise<void>;
  generate: (input: GenerateReportInput) => Promise<GeneratedReport | null>;
  download: (report: GeneratedReport) => Promise<boolean>;
  deleteReport: (id: string) => Promise<boolean>;
}

export function useReportGenerator(): UseReportGeneratorResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [reports, setReports] = useState<GeneratedReport[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchReports = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/reports`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setReports(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const generate = useCallback(async (input: GenerateReportInput): Promise<GeneratedReport | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/reports`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Generate failed (${resp.status})`);
      const report = await resp.json() as GeneratedReport;
      setReports((prev) => [report, ...prev]);
      return report;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const download = useCallback(async (report: GeneratedReport): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}${report.download_url}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Download failed (${resp.status})`);
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `report-${report.framework}-${report.id.slice(0, 8)}.${report.format}`;
      a.click();
      URL.revokeObjectURL(url);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteReport = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/reports/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setReports((prev) => prev.filter((r: any) => r.id !== id));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { reports, isLoading, error, fetchReports, generate, download, deleteReport };
}
