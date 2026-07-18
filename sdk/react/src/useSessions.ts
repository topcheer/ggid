/**
 * GGID React SDK — useSessions hook
 *
 * List and revoke user sessions, view active devices/IPs.
 *
 * Usage:
 *   const { sessions, isLoading, revokeSession, revokeAllOthers, refetch } = useSessions();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface UserSession {
  id: string;
  user_id: string;
  username: string;
  ip_address: string;
  user_agent: string;
  device: 'desktop' | 'mobile' | 'tablet' | 'unknown';
  browser: string;
  os: string;
  location: string;
  created_at: string;
  last_active: string;
  current: boolean;
}

export interface UseSessionsResult {
  sessions: UserSession[];
  isLoading: boolean;
  error: string | null;
  revokeSession: (id: string) => Promise<boolean>;
  revokeAllOthers: () => Promise<number>;
  refetch: () => Promise<void>;
}

export function useSessions(): UseSessionsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [sessions, setSessions] = useState<UserSession[]>([]);
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

  const fetchSessions = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/security/sessions`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch sessions (${resp.status})`);
      const data = await resp.json();
      setSessions(data.sessions ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setSessions([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchSessions();
  }, [isAuthenticated, fetchSessions]);

  const revokeSession = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/security/sessions/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to revoke session (${resp.status})`);
        setSessions((prev) => prev.filter((s: any) => s.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  const revokeAllOthers = useCallback(async (): Promise<number> => {
    const others = sessions.filter((s: any) => !s.current);
    let revoked = 0;
    for (const s of others) {
      const ok = await revokeSession(s.id);
      if (ok) revoked++;
    }
    return revoked;
  }, [sessions, revokeSession]);

  return {
    sessions,
    isLoading,
    error,
    revokeSession,
    revokeAllOthers,
    refetch: fetchSessions,
  };
}
