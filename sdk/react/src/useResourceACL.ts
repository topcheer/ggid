/**
 * GGID React SDK — useResourceACL hook
 *
 * Resource path ACL management: CRUD.
 *
 * Usage:
 *   const { rules, create, update, deleteRule } = useResourceACL();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ResourceACLRule {
  id: string;
  resource_path: string;
  principal: string;
  principal_type: 'user' | 'role' | 'group' | 'service';
  effect: 'allow' | 'deny';
  permissions: string[];
  conditions: string;
  inherited: boolean;
  created_at: string;
}

export interface CreateACLInput {
  resource_path: string;
  principal: string;
  principal_type: ResourceACLRule['principal_type'];
  effect: 'allow' | 'deny';
  permissions?: string[];
  conditions?: string;
}

export interface UseResourceACLResult {
  rules: ResourceACLRule[];
  isLoading: boolean;
  error: string | null;
  fetchRules: (resourcePath?: string) => Promise<void>;
  create: (input: CreateACLInput) => Promise<boolean>;
  update: (id: string, patch: Partial<ResourceACLRule>) => Promise<boolean>;
  deleteRule: (id: string) => Promise<boolean>;
}

export function useResourceACL(): UseResourceACLResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [rules, setRules] = useState<ResourceACLRule[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchRules = useCallback(async (resourcePath?: string) => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = resourcePath ? `?path=${encodeURIComponent(resourcePath)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-acl${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setRules(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const create = useCallback(async (input: CreateACLInput): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-acl`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify(input) });
      if (!resp.ok) throw new Error(`Create failed (${resp.status})`);
      const created = await resp.json() as ResourceACLRule;
      setRules((prev: any) => [...prev, created]);
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const update = useCallback(async (id: string, patch: Partial<ResourceACLRule>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-acl/${id}`, { method: 'PUT', headers: makeHeaders(), body: JSON.stringify(patch) });
      if (!resp.ok) throw new Error(`Update failed (${resp.status})`);
      setRules((prev: any) => prev.map((r) => r.id === id ? { ...r, ...patch } : r));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  const deleteRule = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-acl/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setRules((prev: any) => prev.filter((r) => r.id !== id));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { rules, isLoading, error, fetchRules, create, update, deleteRule };
}
