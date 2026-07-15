/**
 * GGID React SDK — useOrgTree hook
 *
 * Organization tree (LTREE) visualization with lazy child loading.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface OrgTreeNode {
  id: string;
  name: string;
  description: string;
  parent_id?: string;
  path: string;        // LTREE path e.g. "root.acme.engineering"
  depth: number;
  member_count: number;
  children_count: number;
  status: 'active' | 'inactive';
  expanded?: boolean;
  children?: OrgTreeNode[];
}

export interface UseOrgTreeResult {
  tree: OrgTreeNode[];
  isLoading: boolean;
  error: string | null;
  fetchTree: () => Promise<OrgTreeNode[]>;
  fetchChildren: (parentId: string) => Promise<OrgTreeNode[]>;
  searchNodes: (query: string) => Promise<OrgTreeNode[]>;
  getPath: (nodeId: string) => Promise<OrgTreeNode[]>;
  refetch: () => Promise<OrgTreeNode[]>;
}

export function useOrgTree(): UseOrgTreeResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [tree, setTree] = useState<OrgTreeNode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchTree = useCallback(async (): Promise<OrgTreeNode[]> => {
    const tok = getAccessToken();
    if (!tok) return [];
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/tree?depth=1`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch org tree (${resp.status})`);
      const data = await resp.json();
      const nodes = data.tree ?? data.nodes ?? data.orgs ?? [];
      setTree(nodes);
      return nodes;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setTree([]);
      return [];
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  const fetchChildren = useCallback(async (parentId: string): Promise<OrgTreeNode[]> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/${parentId}/children`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch children (${resp.status})`);
      const data = await resp.json();
      return data.children ?? data.nodes ?? [];
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    }
  }, [apiBaseUrl, makeHeaders]);

  const searchNodes = useCallback(async (query: string): Promise<OrgTreeNode[]> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/search?q=${encodeURIComponent(query)}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Search failed (${resp.status})`);
      const data = await resp.json();
      return data.results ?? data.nodes ?? [];
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    }
  }, [apiBaseUrl, makeHeaders]);

  const getPath = useCallback(async (nodeId: string): Promise<OrgTreeNode[]> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/${nodeId}/path`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch path (${resp.status})`);
      const data = await resp.json();
      return data.path ?? data.nodes ?? [];
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    }
  }, [apiBaseUrl, makeHeaders]);

  // Initial load
  useState(() => {
    if (isAuthenticated) fetchTree();
  });

  return {
    tree,
    isLoading,
    error,
    fetchTree,
    fetchChildren,
    searchNodes,
    getPath,
    refetch: fetchTree,
  };
}
