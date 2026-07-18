/**
 * GGID React SDK — usePermissionTree hook
 *
 * Fetch hierarchical permission tree with parent-child relationships.
 *
 * Usage:
 *   const { tree, isLoading, toggleExpand } = usePermissionTree();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface PermissionNode {
  id: string;
  name: string;
  description: string;
  resource: string;
  action: string;
  children: PermissionNode[];
  parent_id: string | null;
}

export interface UsePermissionTreeResult {
  tree: PermissionNode[];
  isLoading: boolean;
  error: string | null;
  expanded: Set<string>;
  toggleExpand: (id: string) => void;
  expandAll: () => void;
  collapseAll: () => void;
  refetch: () => Promise<void>;
}

export function usePermissionTree(): UsePermissionTreeResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [tree, setTree] = useState<PermissionNode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchTree = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/permissions/tree`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch permission tree (${resp.status})`);
      const data = await resp.json();
      setTree(data.tree ?? data.nodes ?? data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setTree([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchTree();
  }, [isAuthenticated, fetchTree]);

  const toggleExpand = useCallback((id: string) => {
    setExpanded((prev: any) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const collectIds = useCallback((nodes: PermissionNode[]): string[] => {
    const ids: string[] = [];
    for (const n of nodes) {
      ids.push(n.id);
      ids.push(...collectIds(n.children));
    }
    return ids;
  }, []);

  const expandAll = useCallback(() => {
    setExpanded(new Set(collectIds(tree)));
  }, [tree, collectIds]);

  const collapseAll = useCallback(() => {
    setExpanded(new Set());
  }, []);

  return {
    tree,
    isLoading,
    error,
    expanded,
    toggleExpand,
    expandAll,
    collapseAll,
    refetch: fetchTree,
  };
}
