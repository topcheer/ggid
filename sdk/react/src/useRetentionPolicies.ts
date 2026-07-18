/**
 * GGID React SDK — useRetentionPolicies hook
 *
 * CRUD for per-event-type data retention policies.
 *
 * Usage:
 *   const { policies, createPolicy, updatePolicy, deletePolicy } = useRetentionPolicies();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type RetentionAction = 'delete' | 'anonymize' | 'archive';

export interface RetentionPolicy {
  id: string;
  event_type: string;
  retention_days: number;
  action: RetentionAction;
  description: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyInput {
  event_type: string;
  retention_days: number;
  action: RetentionAction;
  description?: string;
}

export interface UseRetentionPoliciesResult {
  policies: RetentionPolicy[];
  isLoading: boolean;
  error: string | null;
  createPolicy: (input: CreatePolicyInput) => Promise<RetentionPolicy | null>;
  updatePolicy: (id: string, input: Partial<RetentionPolicy>) => Promise<boolean>;
  deletePolicy: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useRetentionPolicies(): UseRetentionPoliciesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policies, setPolicies] = useState<RetentionPolicy[]>([]);
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

  const fetchPolicies = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention-policies`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch retention policies (${resp.status})`);
      const data = await resp.json();
      setPolicies(data.policies ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setPolicies([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchPolicies();
  }, [isAuthenticated, fetchPolicies]);

  const createPolicy = useCallback(
    async (input: CreatePolicyInput): Promise<RetentionPolicy | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention-policies`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create policy (${resp.status})`);
        const created = await resp.json();
        setPolicies((prev) => [...prev, created]);
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const updatePolicy = useCallback(
    async (id: string, input: Partial<RetentionPolicy>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention-policies/${id}`, {
          method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update policy (${resp.status})`);
        const updated = await resp.json();
        setPolicies((prev) => prev.map((p: any) => (p.id === id ? updated : p)));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const deletePolicy = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/settings/retention-policies/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete policy (${resp.status})`);
        setPolicies((prev) => prev.filter((p: any) => p.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    policies, isLoading, error,
    createPolicy, updatePolicy, deletePolicy,
    refetch: fetchPolicies,
  };
}
