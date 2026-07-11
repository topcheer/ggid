/**
 * GGID React SDK — useComplianceDashboard hook
 *
 * Multi-framework compliance overview.
 *
 * Usage:
 *   const { frameworks, fetchDashboard, isLoading } = useComplianceDashboard();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface FrameworkSummary {
  framework: string;
  coverage_pct: number;
  total_controls: number;
  compliant: number;
  partial: number;
  missing: number;
  gap_count: number;
  last_assessed: string;
}

export interface UseComplianceDashboardResult {
  frameworks: FrameworkSummary[];
  isLoading: boolean;
  error: string | null;
  fetchDashboard: () => Promise<void>;
}

export function useComplianceDashboard(): UseComplianceDashboardResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [frameworks, setFrameworks] = useState<FrameworkSummary[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboard = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/compliance-dashboard`, { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId } });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setFrameworks(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  return { frameworks, isLoading, error, fetchDashboard };
}
