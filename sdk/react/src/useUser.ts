/**
 * GGID React SDK — useUser hook
 *
 * Auto-fetches user profile from GET /api/v1/users/me.
 * Re-fetches when the access token changes.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';
import type { GGIDUser } from './types';

interface UseUserResult {
  user: GGIDUser | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useUser(): UseUserResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const token = getAccessToken();

  const [user, setUser] = useState<GGIDUser | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Infer config from context
  const { tokenSet } = useGGIDAuth();

  // We need the apiBaseUrl and tenantId from the provider config
  // Since useGGIDAuth doesn't expose config, we use the token to call
  // a relative path or the configured base URL stored in localStorage
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const fetchUser = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users/me`, {
        headers: {
          Authorization: `Bearer ${tok}`,
          'X-Tenant-ID': tenantId,
        },
      });
      if (!resp.ok) throw new Error('Failed to fetch user');
      const data = await resp.json();
      setUser(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  useEffect(() => {
    if (isAuthenticated && token) {
      fetchUser();
    }
  }, [isAuthenticated, token, fetchUser]);

  return { user, isLoading, error, refresh: fetchUser };
}
