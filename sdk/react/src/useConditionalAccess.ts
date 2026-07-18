/**
 * GGID React SDK — useConditionalAccess hook
 *
 * CRUD for conditional access policies.
 *
 * Usage:
 *   const { policies, createPolicy, updatePolicy, deletePolicy } = useConditionalAccess();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type AccessAction = 'allow' | 'deny' | 'require_mfa';

export interface ConditionalAccessPolicy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  conditions: {
    ip_ranges?: string[];
    time_window?: { start: string; end: string };
    device_trusted?: boolean;
    min_risk_score?: number;
    countries?: string[];
  };
  action: AccessAction;
  priority: number;
  created_at: string;
}

export interface CreatePolicyInput {
  name: string;
  description?: string;
  conditions: ConditionalAccessPolicy['conditions'];
  action: AccessAction;
  priority?: number;
}

export interface UseConditionalAccessResult {
  policies: ConditionalAccessPolicy[];
  isLoading: boolean;
  error: string | null;
  createPolicy: (input: CreatePolicyInput) => Promise<ConditionalAccessPolicy | null>;
  updatePolicy: (id: string, input: Partial<ConditionalAccessPolicy>) => Promise<boolean>;
  deletePolicy: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useConditionalAccess(): UseConditionalAccessResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policies, setPolicies] = useState<ConditionalAccessPolicy[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchPolicies = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/conditional-access`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      const data = await resp.json();
      setPolicies(data.policies ?? data.items ?? []);
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); setPolicies([]); }
    finally { setIsLoading(false); }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => { if (isAuthenticated) fetchPolicies(); }, [isAuthenticated, fetchPolicies]);

  const createPolicy = useCallback(async (input: CreatePolicyInput): Promise<ConditionalAccessPolicy | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/conditional-access`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Create failed (${resp.status})`);
      const created = await resp.json(); await fetchPolicies(); return created;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders, fetchPolicies]);

  const updatePolicy = useCallback(async (id: string, input: Partial<ConditionalAccessPolicy>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/conditional-access/${id}`, { method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      await fetchPolicies(); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchPolicies]);

  const deletePolicy = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/conditional-access/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setPolicies((prev: any) => prev.filter((p) => p.id !== id)); return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { policies, isLoading, error, createPolicy, updatePolicy, deletePolicy, refetch: fetchPolicies };
}
