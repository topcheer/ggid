/**
 * GGID React SDK — useIdentityGovernance hook
 *
 * Identity Governance & Administration metrics.
 *
 * Usage:
 *   const { metrics, isLoading } = useIdentityGovernance();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface IGAMetrics {
  open_campaigns: number;
  pending_reviews: number;
  overdue_reviews: number;
  sod_violations: { critical: number; high: number; medium: number };
  orphaned_accounts: number;
  dormant_accounts: number;
  cert_completion_rate: number;
  avg_review_time_hours: number;
  recent_campaigns: { id: string; name: string; status: string; completion: number }[];
}

export interface UseIdentityGovernanceResult {
  metrics: IGAMetrics | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useIdentityGovernance(): UseIdentityGovernanceResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [metrics, setMetrics] = useState<IGAMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMetrics = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/iga/metrics`, {
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId },
      });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setMetrics(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  return { metrics, isLoading, error, refetch: fetchMetrics };
}
