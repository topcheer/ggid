/**
 * GGID React SDK — usePolicies hook
 *
 * Policy CRUD + versioning + ABAC rule management.
 *
 * Usage:
 *   const { policies, isLoading, createPolicy, updatePolicy, deletePolicy, refetch } = usePolicies();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ABACRule {
  attribute: string;
  operator: 'eq' | 'ne' | 'in' | 'not_in' | 'gt' | 'lt' | 'gte' | 'lte' | 'contains';
  value: string | number | boolean | string[];
}

export interface Policy {
  id: string;
  name: string;
  description: string;
  effect: 'allow' | 'deny';
  resource: string;
  action: string;
  conditions: ABACRule[];
  version: number;
  enabled: boolean;
  priority: number;
  created_at: string;
  updated_at?: string;
}

export interface CreatePolicyInput {
  name: string;
  description?: string;
  effect: 'allow' | 'deny';
  resource: string;
  action: string;
  conditions?: ABACRule[];
  priority?: number;
}

export interface UpdatePolicyInput {
  name?: string;
  description?: string;
  effect?: 'allow' | 'deny';
  resource?: string;
  action?: string;
  conditions?: ABACRule[];
  enabled?: boolean;
  priority?: number;
}

export interface UsePoliciesResult {
  policies: Policy[];
  isLoading: boolean;
  error: string | null;
  createPolicy: (input: CreatePolicyInput) => Promise<Policy | null>;
  updatePolicy: (id: string, input: UpdatePolicyInput) => Promise<boolean>;
  deletePolicy: (id: string) => Promise<boolean>;
  getPolicyVersions: (id: string) => Promise<Policy[]>;
  refetch: () => Promise<void>;
}

export function usePolicies(): UsePoliciesResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policies, setPolicies] = useState<Policy[]>([]);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/policies`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch policies (${resp.status})`);
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
    async (input: CreatePolicyInput): Promise<Policy | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policies`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create policy (${resp.status})`);
        const created = await resp.json();
        await fetchPolicies();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchPolicies]
  );

  const updatePolicy = useCallback(
    async (id: string, input: UpdatePolicyInput): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policies/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update policy (${resp.status})`);
        await fetchPolicies();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchPolicies]
  );

  const deletePolicy = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policies/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete policy (${resp.status})`);
        await fetchPolicies();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchPolicies]
  );

  const getPolicyVersions = useCallback(
    async (id: string): Promise<Policy[]> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policies/${id}/versions`, {
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to fetch policy versions (${resp.status})`);
        const data = await resp.json();
        return data.versions ?? data.policies ?? [];
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return [];
      }
    },
    [apiBaseUrl, makeHeaders]
  );

  return {
    policies,
    isLoading,
    error,
    createPolicy,
    updatePolicy,
    deletePolicy,
    getPolicyVersions,
    refetch: fetchPolicies,
  };
}
