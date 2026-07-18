/**
 * GGID React SDK — useConsent hook
 *
 * OAuth consent management: list granted consents, revoke by client.
 *
 * Usage:
 *   const { consents, revokeConsent, isLoading } = useConsent();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Consent {
  id: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  granted_at: string;
  last_used?: string;
  status: 'active' | 'revoked';
}

export interface UseConsentResult {
  consents: Consent[];
  isLoading: boolean;
  error: string | null;
  revokeConsent: (id: string) => Promise<boolean>;
  revokeByClient: (clientId: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useConsent(): UseConsentResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [consents, setConsents] = useState<Consent[]>([]);
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

  const fetchConsents = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/consents`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch consents (${resp.status})`);
      const data = await resp.json();
      setConsents(data.consents ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setConsents([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchConsents();
  }, [isAuthenticated, fetchConsents]);

  const revokeConsent = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/consents/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to revoke consent (${resp.status})`);
        setConsents((prev: any) => prev.filter((c: any) => c.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const revokeByClient = useCallback(
    async (clientId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/consents?client_id=${clientId}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to revoke consent (${resp.status})`);
        setConsents((prev: any) => prev.filter((c: any) => c.client_id !== clientId));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    consents,
    isLoading,
    error,
    revokeConsent,
    revokeByClient,
    refetch: fetchConsents,
  };
}
