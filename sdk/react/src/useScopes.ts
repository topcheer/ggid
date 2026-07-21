/**
 * GGID React SDK — useScopes hook
 *
 * OAuth scope CRUD with wildcard expansion.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface OAuthScope {
  id: string;
  name: string;
  description: string;
  is_wildcard: boolean;
  parent_scope?: string;
  resource_type: string;
  actions: string[];
  default: boolean;
  created_at: string;
  updated_at?: string;
}

export interface CreateScopeInput {
  name: string;
  description?: string;
  resource_type?: string;
  actions?: string[];
  parent_scope?: string;
}

export interface UpdateScopeInput {
  name?: string;
  description?: string;
  actions?: string[];
  parent_scope?: string;
}

export interface UseScopesResult {
  scopes: OAuthScope[];
  isLoading: boolean;
  error: string | null;
  createScope: (input: CreateScopeInput) => Promise<OAuthScope | null>;
  updateScope: (id: string, input: UpdateScopeInput) => Promise<boolean>;
  deleteScope: (id: string) => Promise<boolean>;
  expandWildcard: (wildcard: string) => OAuthScope[];
  refetch: () => Promise<void>;
}

export function useScopes(): UseScopesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [scopes, setScopes] = useState<OAuthScope[]>([]);
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

  const fetchScopes = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/scopes`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch scopes (${resp.status})`);
      const data = await resp.json();
      setScopes(data.scopes ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setScopes([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchScopes();
  }, [isAuthenticated, fetchScopes]);

  const createScope = useCallback(
    async (input: CreateScopeInput): Promise<OAuthScope | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scopes`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create scope (${resp.status})`);
        const created = await resp.json();
        await fetchScopes();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchScopes]
  );

  const updateScope = useCallback(
    async (id: string, input: UpdateScopeInput): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scopes/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update scope (${resp.status})`);
        await fetchScopes();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchScopes]
  );

  const deleteScope = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scopes/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete scope (${resp.status})`);
        await fetchScopes();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchScopes]
  );

  /** Expand a wildcard scope (e.g. "users:*" → all scopes starting with "users:") */
  const expandWildcard = useCallback(
    (wildcard: string): OAuthScope[] => {
      if (!wildcard.includes('*')) return [];
      const prefix = wildcard.slice(0, wildcard.indexOf('*'));
      return scopes.filter((s: any) => s.name.startsWith(prefix));
    },
    [scopes]
  );

  return {
    scopes,
    isLoading,
    error,
    createScope,
    updateScope,
    deleteScope,
    expandWildcard,
    refetch: fetchScopes,
  };
}
