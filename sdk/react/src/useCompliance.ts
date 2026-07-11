/**
 * GGID React SDK — useCompliance hook
 *
 * Fetch SOC2/HIPAA/GDPR compliance reports with date range.
 *
 * Usage:
 *   const { reports, isLoading, downloadReport, refetch } = useCompliance({
 *     framework: 'soc2',
 *     dateFrom: '2025-01-01',
 *     dateTo: '2025-06-30',
 *   });
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type ComplianceFramework = 'soc2' | 'hipaa' | 'gdpr' | 'iso27001' | 'pci';

export interface ComplianceReport {
  id: string;
  framework: string;
  status: 'compliant' | 'partial' | 'non-compliant';
  score: number;
  last_assessed: string;
  controls_total: number;
  controls_passed: number;
  controls_failed: number;
  summary: string;
}

export interface ComplianceFilter {
  framework?: ComplianceFramework;
  dateFrom?: string;
  dateTo?: string;
}

export interface UseComplianceResult {
  reports: ComplianceReport[];
  isLoading: boolean;
  error: string | null;
  downloadReport: (id: string, format: 'pdf' | 'csv') => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useCompliance(filter: ComplianceFilter = {}): UseComplianceResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [reports, setReports] = useState<ComplianceReport[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchReports = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (filter.framework) params.set('framework', filter.framework);
      if (filter.dateFrom) params.set('date_from', filter.dateFrom);
      if (filter.dateTo) params.set('date_to', filter.dateTo);

      const resp = await fetch(
        `${apiBaseUrl}/api/v1/audit/compliance/reports?${params.toString()}`,
        { headers: makeHeaders() }
      );
      if (!resp.ok) throw new Error(`Failed to fetch compliance reports (${resp.status})`);
      const data = await resp.json();
      setReports(data.reports ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setReports([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders,
      filter.framework, filter.dateFrom, filter.dateTo]);

  useEffect(() => {
    if (isAuthenticated) fetchReports();
  }, [isAuthenticated, fetchReports]);

  const downloadReport = useCallback(
    async (id: string, format: 'pdf' | 'csv'): Promise<boolean> => {
      try {
        const resp = await fetch(
          `${apiBaseUrl}/api/v1/audit/compliance/reports/${id}/download?format=${format}`,
          { headers: makeHeaders() }
        );
        if (!resp.ok) throw new Error(`Failed to download report (${resp.status})`);
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${id}.${format}`;
        a.click();
        URL.revokeObjectURL(url);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    reports,
    isLoading,
    error,
    downloadReport,
    refetch: fetchReports,
  };
}
