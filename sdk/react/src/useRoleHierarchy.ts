/**
 * GGID React SDK — useRoleHierarchy hook
 *
 * Fetch role inheritance tree with recursive expand/collapse.
 *
 * Usage:
 *   const { tree, isLoading } = useRoleHierarchy();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RoleNode {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  inherits_from: string | null;
  children: RoleNode[];
  user_count: number;
}

export interface UseRoleHierarchyResult {
  tree: RoleNode[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useRoleHierarchy(): UseRoleHierarchyResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [tree, setTree] = useState<RoleNode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTree = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/roles/hierarchy`, {
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId },
      });
      if (!resp.ok) throw new Error(`Failed to fetch role hierarchy (${resp.status})`);
      const data = await resp.json();
      setTree(data.tree ?? data.nodes ?? data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setTree([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, tenantId]);

  useEffect(() => {
    if (isAuthenticated) fetchTree();
  }, [isAuthenticated, fetchTree]);

  return { tree, isLoading, error, refetch: fetchTree };
}
