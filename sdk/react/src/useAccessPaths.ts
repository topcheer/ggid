/**
 * GGID React SDK — useAccessPaths hook
 *
 * User privilege path analysis.
 *
 * Usage:
 *   const { paths, analyze, isLoading } = useAccessPaths();
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface AccessPathNode {
  resource: string;
  resource_type: string;
  access_level: string;
  source: string;
  over_privileged: boolean;
  children: AccessPathNode[];
}

export interface AccessPathResult {
  user_id: string;
  username: string;
  total_resources: number;
  over_privileged_count: number;
  paths: AccessPathNode[];
}

export interface UseAccessPathsResult {
  result: AccessPathResult | null;
  isLoading: boolean;
  error: string | null;
  analyze: (userId: string) => Promise<void>;
}

export function useAccessPaths(): UseAccessPathsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [result, setResult] = useState<AccessPathResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const analyze = useCallback(async (userId: string) => {
    setIsLoading(true); setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/access-paths?user_id=${encodeURIComponent(userId)}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Analysis failed (${resp.status})`);
      setResult(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  return { result, isLoading, error, analyze };
}
