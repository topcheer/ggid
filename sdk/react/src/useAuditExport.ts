/**
 * GGID React SDK — useAuditExport hook
 *
 * Export audit events as CSV/JSON with filters.
 *
 * Usage:
 *   const { exportEvents, isExporting } = useAuditExport();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ExportParams {
  format: 'csv' | 'json';
  date_from?: string;
  date_to?: string;
  event_type?: string;
  tenant_id?: string;
}

export interface UseAuditExportResult {
  isExporting: boolean;
  error: string | null;
  exportEvents: (params: ExportParams) => Promise<boolean>;
}

export function useAuditExport(): UseAuditExportResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const exportEvents = useCallback(
    async (params: ExportParams): Promise<boolean> => {
      const tok = getAccessToken();
      if (!tok) return false;
      setIsExporting(true);
      setError(null);
      try {
        const searchParams = new URLSearchParams();
        searchParams.set('format', params.format);
        if (params.date_from) searchParams.set('date_from', params.date_from);
        if (params.date_to) searchParams.set('date_to', params.date_to);
        if (params.event_type) searchParams.set('event_type', params.event_type);
        searchParams.set('tenant_id', params.tenant_id || tenantId);

        const resp = await fetch(`${apiBaseUrl}/api/v1/audit/export?${searchParams.toString()}`, {
          headers: {
            Authorization: `Bearer ${tok}`,
            'X-Tenant-ID': params.tenant_id || tenantId,
          },
        });
        if (!resp.ok) throw new Error(`Export failed (${resp.status})`);
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        const ts = new Date().toISOString().split('T')[0];
        a.href = url;
        a.download = `audit-export-${ts}.${params.format}`;
        a.click();
        URL.revokeObjectURL(url);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      } finally {
        setIsExporting(false);
      }
    },
    [apiBaseUrl, getAccessToken, tenantId],
  );

  return { isExporting, error, exportEvents };
}
