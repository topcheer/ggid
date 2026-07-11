/**
 * GGID React SDK — usePasswordBreach hook
 *
 * HIBP breach monitoring and notification.
 *
 * Usage:
 *   const { status, notifyAffected } = usePasswordBreach();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface BreachStatus {
  hibp_enabled: boolean;
  last_check: string;
  total_breaches: number;
  affected_users: number;
  notified_users: number;
  breaches: { name: string; date: string; affected_count: number; notified: boolean }[];
}

export interface UsePasswordBreachResult {
  status: BreachStatus | null;
  isLoading: boolean;
  error: string | null;
  notifyAffected: (breachName: string) => Promise<boolean>;
  fetchStatus: () => Promise<void>;
}

export function usePasswordBreach(): UsePasswordBreachResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [status, setStatus] = useState<BreachStatus | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchStatus = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/password-breach/status`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setStatus(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const notifyAffected = useCallback(async (breachName: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/password-breach/notify`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ breach_name: breachName }) });
      if (!resp.ok) throw new Error(`Notify failed (${resp.status})`);
      await fetchStatus(); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchStatus]);

  return { status, isLoading, error, notifyAffected, fetchStatus };
}
