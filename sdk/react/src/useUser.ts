/**
 * GGID React SDK — useUser hook
 * Auto-fetches GET /api/v1/users/me
 */

import { useState, useEffect } from 'react';
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
  const [user, setUser] = useState<GGIDUser | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUser = async () => {
    const token = getAccessToken();
    if (!token) return;
    setIsLoading(true);
    setError(null);
    try {
      // apiBaseUrl and tenantId are inferred from the GGIDProvider context
      // but useGGIDAuth doesn't expose them — use the token directly
      const resp = await fetch('/api/v1/users/me', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!resp.ok) throw new Error('Failed to fetch user');
      const data = await resp.json();
      setUser(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (isAuthenticated) {
      fetchUser();
    }
  }, [isAuthenticated]); // eslint-disable-line react-hooks/exhaustive-deps

  return { user, isLoading, error, refresh: fetchUser };
}
