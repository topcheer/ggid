/**
 * GGID React SDK — useImpersonationLog hook
 *
 * Fetch impersonation audit trail with filters.
 *
 * Usage:
 *   const { entries, isLoading } = useImpersonationLog({ impersonator: 'admin' });
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ImpersonationEntry {
  id: string;
  impersonator_id: string;
  impersonator_name: string;
  target_id: string;
  target_name: string;
  started_at: string;
  ended_at: string | null;
  duration_seconds: number;
  ip_address: string;
  reason: string;
  actions_taken: number;
}

export interface ImpersonationFilter {
  impersonator?: string;
  target?: string;
  date_from?: string;
  date_to?: string;
}

export interface UseImpersonationLogResult {
  entries: ImpersonationEntry[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useImpersonationLog(filter: ImpersonationFilter = {}): UseImpersonationLogResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [entries, setEntries] = useState<ImpersonationEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEntries = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (filter.impersonator) params.set('impersonator', filter.impersonator);
      if (filter.target) params.set('target', filter.target);
      if (filter.date_from) params.set('date_from', filter.date_from);
      if (filter.date_to) params.set('date_to', filter.date_to);
      const resp = await fetch(`${apiBaseUrl}/api/v1/audit/impersonation?${params.toString()}`, {
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId },
      });
      if (!resp.ok) throw new Error(`Failed to fetch impersonation log (${resp.status})`);
      const data = await resp.json();
      setEntries(data.entries ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId, filter.impersonator, filter.target, filter.date_from, filter.date_to]);

  useEffect(() => {
    if (isAuthenticated) fetchEntries();
  }, [isAuthenticated, fetchEntries]);

  return { entries, isLoading, error, refetch: fetchEntries };
}
