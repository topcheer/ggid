/**
 * GGID React SDK — useSoDMatrix hook
 *
 * Separation of Duties role exclusion matrix.
 *
 * Usage:
 *   const { matrix, rules, toggleExclusion, fetchMatrix } = useSoDMatrix();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface SoDRule {
  id: string;
  role_a: string;
  role_b: string;
  reason: string;
  created_at: string;
}

export interface UseSoDMatrixResult {
  matrix: boolean[][];
  roles: string[];
  rules: SoDRule[];
  isLoading: boolean;
  error: string | null;
  fetchMatrix: () => Promise<void>;
  toggleExclusion: (roleA: string, roleB: string) => Promise<boolean>;
}

export function useSoDMatrix(): UseSoDMatrixResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [matrix, setMatrix] = useState<boolean[][]>([]);
  const [roles, setRoles] = useState<string[]>([]);
  const [rules, setRules] = useState<SoDRule[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchMatrix = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod-matrix`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      const data = await resp.json();
      setMatrix(data.matrix || []);
      setRoles(data.roles || []);
      setRules(data.rules || []);
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const toggleExclusion = useCallback(async (roleA: string, roleB: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/sod-matrix/toggle`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ role_a: roleA, role_b: roleB }) });
      if (!resp.ok) throw new Error(`Toggle failed (${resp.status})`);
      await fetchMatrix();
      return true;
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchMatrix]);

  return { matrix, roles, rules, isLoading, error, fetchMatrix, toggleExclusion };
}
