/**
 * GGID React SDK — useLoginAttempts hook
 *
 * Track failed login attempts and manage account lockouts.
 *
 * Usage:
 *   const { attempts, lockouts, resetAttempts, unlock } = useLoginAttempts();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface LoginAttempt {
  id: string;
  user_id: string;
  user_name: string;
  ip_address: string;
  success: boolean;
  timestamp: string;
  user_agent?: string;
  failure_reason?: string;
}

export interface Lockout {
  id: string;
  user_id: string;
  user_name: string;
  failed_count: number;
  locked_until: string;
  locked_at: string;
  ip_address: string;
}

export interface LoginPolicy {
  max_attempts: number;
  lockout_duration_minutes: number;
}

export interface UseLoginAttemptsResult {
  attempts: LoginAttempt[];
  lockouts: Lockout[];
  policy: LoginPolicy | null;
  isLoading: boolean;
  error: string | null;
  resetAttempts: (userId: string) => Promise<boolean>;
  unlock: (userId: string) => Promise<boolean>;
  updatePolicy: (policy: Partial<LoginPolicy>) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useLoginAttempts(): UseLoginAttemptsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [attempts, setAttempts] = useState<LoginAttempt[]>([]);
  const [lockouts, setLockouts] = useState<Lockout[]>([]);
  const [policy, setPolicy] = useState<LoginPolicy | null>(null);
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

  const fetchAll = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const [attRes, lockRes, polRes] = await Promise.all([
        fetch(`${apiBaseUrl}/api/v1/auth/login-attempts?limit=50`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/auth/lockouts`, { headers: makeHeaders() }),
        fetch(`${apiBaseUrl}/api/v1/auth/login-policy`, { headers: makeHeaders() }),
      ]);
      if (attRes.ok) {
        const d = await attRes.json();
        setAttempts(d.attempts ?? d.items ?? []);
      }
      if (lockRes.ok) {
        const d = await lockRes.json();
        setLockouts(d.lockouts ?? d.items ?? []);
      }
      if (polRes.ok) setPolicy(await polRes.json());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchAll();
  }, [isAuthenticated, fetchAll]);

  const resetAttempts = useCallback(
    async (userId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/auth/login-attempts/reset`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify({ user_id: userId }),
        });
        if (!resp.ok) throw new Error(`Failed to reset attempts (${resp.status})`);
        await fetchAll();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchAll],
  );

  const unlock = useCallback(
    async (userId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/auth/lockouts/${userId}/unlock`, {
          method: 'POST', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to unlock (${resp.status})`);
        setLockouts((prev: any) => prev.filter((l) => l.user_id !== userId));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const updatePolicy = useCallback(
    async (p: Partial<LoginPolicy>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/auth/login-policy`, {
          method: 'PUT', headers: makeHeaders(), body: JSON.stringify(p),
        });
        if (!resp.ok) throw new Error(`Failed to update policy (${resp.status})`);
        const updated = await resp.json();
        setPolicy(updated);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    attempts, lockouts, policy, isLoading, error,
    resetAttempts, unlock, updatePolicy, refetch: fetchAll,
  };
}
