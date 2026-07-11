/**
 * GGID React SDK — useRetention hook
 *
 * Fetch and update audit log retention policy.
 *
 * Usage:
 *   const { policy, isLoading, updatePolicy, refetch } = useRetention();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RetentionPolicy {
  max_age_days: number;
  max_events: number;
  archive_enabled: boolean;
  archive_location?: string;
  delete_archived: boolean;
  compliance_mode: boolean;
}

export interface UseRetentionResult {
  policy: RetentionPolicy | null;
  isLoading: boolean;
  error: string | null;
  updatePolicy: (policy: Partial<RetentionPolicy>) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useRetention(): UseRetentionResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policy, setPolicy] = useState<RetentionPolicy | null>(null);
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

  const fetchPolicy = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch retention policy (${resp.status})`);
      const data = await resp.json();
      setPolicy(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setPolicy(null);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchPolicy();
  }, [isAuthenticated, fetchPolicy]);

  const updatePolicy = useCallback(
    async (newPolicy: Partial<RetentionPolicy>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention`, {
          method: 'PUT',
          headers: makeHeaders(),
          body: JSON.stringify(newPolicy),
        });
        if (!resp.ok) throw new Error(`Failed to update retention policy (${resp.status})`);
        const updated = await resp.json();
        setPolicy(updated);
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    policy,
    isLoading,
    error,
    updatePolicy,
    refetch: fetchPolicy,
  };
}
