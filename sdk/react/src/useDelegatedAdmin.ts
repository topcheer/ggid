/**
 * GGID React SDK — useDelegatedAdmin hook
 *
 * Delegated administration: grant/revoke scoped admin access.
 *
 * Usage:
 *   const { delegations, grant, revoke, isLoading } = useDelegatedAdmin();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export type DelegationScope = 'org' | 'role' | 'dept' | 'global';
export type DelegationPermission = 'read' | 'write' | 'admin' | 'full';

export interface Delegation {
  id: string;
  delegate: string;
  delegate_name: string;
  scope_type: DelegationScope;
  scope_value: string;
  permissions: DelegationPermission[];
  granted_by: string;
  granted_at: string;
  expires_at: string;
  revoked: boolean;
}

export interface GrantDelegationInput {
  delegate: string;
  scope_type: DelegationScope;
  scope_value: string;
  permissions: DelegationPermission[];
  expires_at: string;
}

export interface UseDelegatedAdminResult {
  delegations: Delegation[];
  isLoading: boolean;
  error: string | null;
  fetchDelegations: (activeOnly?: boolean) => Promise<void>;
  grant: (input: GrantDelegationInput) => Promise<boolean>;
  revoke: (id: string) => Promise<boolean>;
}

export function useDelegatedAdmin(): UseDelegatedAdminResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchDelegations = useCallback(async (activeOnly = false) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = activeOnly ? '?active=true' : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/delegated-admin${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setDelegations(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const grant = useCallback(async (input: GrantDelegationInput): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/delegated-admin`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Grant failed (${resp.status})`);
      const created = await resp.json() as Delegation;
      setDelegations((prev) => [created, ...prev]);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const revoke = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/delegated-admin/${id}/revoke`, { method: 'POST', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Revoke failed (${resp.status})`);
      setDelegations((prev) => prev.map((d: any) => d.id === id ? { ...d, revoked: true } : d));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { delegations, isLoading, error, fetchDelegations, grant, revoke };
}
