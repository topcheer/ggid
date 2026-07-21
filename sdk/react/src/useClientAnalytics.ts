/**
 * GGID React SDK — useClientAnalytics hook
 *
 * OAuth client usage analytics.
 *
 * Usage:
 *   const { analytics, fetchAnalytics, isLoading } = useClientAnalytics();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ClientAnalyticsEntry {
  client_id: string;
  client_name: string;
  total_tokens: number;
  active_tokens: number;
  unique_users: number;
  error_rate: number;
  total_requests: number;
  top_scopes: { scope: string; count: number }[];
  last_active: string;
}

export interface UseClientAnalyticsResult {
  analytics: ClientAnalyticsEntry[];
  isLoading: boolean;
  error: string | null;
  fetchAnalytics: (clientId?: string) => Promise<void>;
}

export function useClientAnalytics(): UseClientAnalyticsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [analytics, setAnalytics] = useState<ClientAnalyticsEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchAnalytics = useCallback(async (clientId?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = clientId ? `?client_id=${encodeURIComponent(clientId)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/analytics${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setAnalytics(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  return { analytics, isLoading, error, fetchAnalytics };
}
