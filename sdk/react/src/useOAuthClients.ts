/**
 * GGID React SDK — useOAuthClients hook
 *
 * OAuth client CRUD + secret regeneration.
 *
 * Usage:
 *   const { clients, isLoading, createClient, updateClient, deleteClient, regenerateSecret, refetch } = useOAuthClients();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface OAuthClient {
  id: string;
  client_id: string;
  client_name: string;
  redirect_uris: string[];
  grant_types: string[];
  scopes: string[];
  token_endpoint_auth_method: string;
  enabled: boolean;
  created_at: string;
  updated_at?: string;
}

export interface CreateOAuthClientInput {
  client_name: string;
  redirect_uris: string[];
  grant_types?: string[];
  scopes?: string[];
  token_endpoint_auth_method?: string;
}

export interface UseOAuthClientsResult {
  clients: OAuthClient[];
  isLoading: boolean;
  error: string | null;
  createClient: (input: CreateOAuthClientInput) => Promise<{ client: OAuthClient; client_secret: string } | null>;
  updateClient: (id: string, input: Partial<OAuthClient>) => Promise<boolean>;
  deleteClient: (id: string) => Promise<boolean>;
  regenerateSecret: (id: string) => Promise<string | null>;
  refetch: () => Promise<void>;
}

export function useOAuthClients(): UseOAuthClientsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [clients, setClients] = useState<OAuthClient[]>([]);
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

  const fetchClients = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/clients`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch OAuth clients (${resp.status})`);
      const data = await resp.json();
      setClients(data.clients ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setClients([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchClients();
  }, [isAuthenticated, fetchClients]);

  const createClient = useCallback(
    async (input: CreateOAuthClientInput): Promise<{ client: OAuthClient; client_secret: string } | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/clients`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create OAuth client (${resp.status})`);
        const data = await resp.json();
        await fetchClients();
        return { client: data.client ?? data, client_secret: data.client_secret ?? '' };
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchClients]
  );

  const updateClient = useCallback(
    async (id: string, input: Partial<OAuthClient>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/clients/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update OAuth client (${resp.status})`);
        await fetchClients();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchClients]
  );

  const deleteClient = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/clients/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete OAuth client (${resp.status})`);
        await fetchClients();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchClients]
  );

  const regenerateSecret = useCallback(
    async (id: string): Promise<string | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/oauth/clients/${id}/regenerate-secret`, {
          method: 'POST',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to regenerate secret (${resp.status})`);
        const data = await resp.json();
        return data.client_secret ?? null;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    clients,
    isLoading,
    error,
    createClient,
    updateClient,
    deleteClient,
    regenerateSecret,
    refetch: fetchClients,
  };
}
