/**
 * GGID React SDK — useDelegation hook
 *
 * Delegate roles to other users with automatic expiry.
 *
 * Usage:
 *   const { delegations, delegate, revoke, isLoading } = useDelegation();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Delegation {
  id: string;
  delegator_id: string;
  delegator_name: string;
  delegate_id: string;
  delegate_name: string;
  roles: string[];
  scope: string;
  created_at: string;
  expires_at: string;
  status: 'active' | 'expired' | 'revoked';
  last_used?: string;
}

export interface DelegateInput {
  delegate_id: string;
  roles: string[];
  scope?: string;
  expires_hours?: number;
}

export interface UseDelegationResult {
  delegations: Delegation[];
  isLoading: boolean;
  error: string | null;
  delegate: (input: DelegateInput) => Promise<Delegation | null>;
  revoke: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useDelegation(): UseDelegationResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [delegations, setDelegations] = useState<Delegation[]>([]);
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

  const fetchDelegations = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/delegations`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch delegations (${resp.status})`);
      const data = await resp.json();
      setDelegations(data.delegations ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setDelegations([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchDelegations();
  }, [isAuthenticated, fetchDelegations]);

  const delegate = useCallback(
    async (input: DelegateInput): Promise<Delegation | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/delegations`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create delegation (${resp.status})`);
        const created = await resp.json();
        await fetchDelegations();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchDelegations],
  );

  const revoke = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/delegations/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to revoke delegation (${resp.status})`);
        setDelegations((prev) => prev.filter((d: any) => d.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    delegations,
    isLoading,
    error,
    delegate,
    revoke,
    refetch: fetchDelegations,
  };
}
