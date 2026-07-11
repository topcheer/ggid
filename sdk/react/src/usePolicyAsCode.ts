/**
 * GGID React SDK — usePolicyAsCode hook
 *
 * YAML policy import/export/preview.
 *
 * Usage:
 *   const { policies, importYaml, previewDiff, exportYaml } = usePolicyAsCode();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface PolicyFile {
  id: string;
  name: string;
  version: string;
  yaml: string;
  status: 'active' | 'draft' | 'archived';
  updated_at: string;
  diff: string;
}

export interface UsePolicyAsCodeResult {
  policies: PolicyFile[];
  isLoading: boolean;
  error: string | null;
  fetchPolicies: () => Promise<void>;
  importYaml: (yaml: string, name: string) => Promise<boolean>;
  previewDiff: (id: string, yaml: string) => Promise<string | null>;
  exportYaml: (id: string) => Promise<string | null>;
  deletePolicy: (id: string) => Promise<boolean>;
}

export function usePolicyAsCode(): UsePolicyAsCodeResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [policies, setPolicies] = useState<PolicyFile[]>([]);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/as-code`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setPolicies(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const importYaml = useCallback(async (yaml: string, name: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/as-code/import`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ yaml, name }) });
      if (!resp.ok) throw new Error(`Import failed (${resp.status})`);
      await fetchPolicies();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchPolicies]);

  const previewDiff = useCallback(async (id: string, yaml: string): Promise<string | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/as-code/${id}/diff`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ yaml }) });
      if (!resp.ok) throw new Error(`Diff failed (${resp.status})`);
      const data = await resp.json();
      return data.diff || '';
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const exportYaml = useCallback(async (id: string): Promise<string | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/as-code/${id}/export`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Export failed (${resp.status})`);
      const data = await resp.json();
      return data.yaml || '';
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return null; }
  }, [apiBaseUrl, makeHeaders]);

  const deletePolicy = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/as-code/${id}`, { method: 'DELETE', headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Delete failed (${resp.status})`);
      setPolicies((prev) => prev.filter((p) => p.id !== id));
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { policies, isLoading, error, fetchPolicies, importYaml, previewDiff, exportYaml, deletePolicy };
}
