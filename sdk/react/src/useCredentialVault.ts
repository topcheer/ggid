/**
 * GGID React SDK — useCredentialVault hook
 *
 * Secure credential storage: store/get/delete.
 *
 * Usage:
 *   const { credentials, store, reveal, remove, isLoading } = useCredentialVault();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface StoredCredential {
  id: string;
  key: string;
  description: string;
  created_at: string;
  last_accessed: string;
  access_count: number;
  expires_at: string;
}

export interface UseCredentialVaultResult {
  credentials: StoredCredential[];
  isLoading: boolean;
  error: string | null;
  fetchCredentials: () => Promise<void>;
  store: (key: string, value: string, description?: string) => Promise<boolean>;
  reveal: (id: string) => Promise<string | null>;
  remove: (id: string) => Promise<boolean>;
}

export function useCredentialVault(): UseCredentialVaultResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [credentials, setCredentials] = useState<StoredCredential[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchCredentials = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/credential-vault`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setCredentials(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const store = useCallback(async (key: string, value: string, description?: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/credential-vault`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ key, value, description }) });
      if (!resp.ok) throw new Error(`Store failed (${resp.status})`);
      await fetchCredentials();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchCredentials]);

  const reveal = useCallback(async (id: string): Promise<string | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/credential-vault/${id}/reveal`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Reveal failed (${resp.status})`);
      const data = await resp.json();
      return data.value || null;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const remove = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/auth/credential-vault/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setCredentials((prev) => prev.filter((c) => c.id !== id));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { credentials, isLoading, error, fetchCredentials, store, reveal, remove };
}
