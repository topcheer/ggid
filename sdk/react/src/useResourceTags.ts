/**
 * GGID React SDK — useResourceTags hook
 *
 * Resource tag management: CRUD + bulk assign.
 */

import { useState, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ResourceTag {
  id: string;
  resource_path: string;
  resource_type: string;
  tags: Record<string, string>;
  updated_at: string;
}

export interface UseResourceTagsResult {
  tags: ResourceTag[];
  isLoading: boolean;
  error: string | null;
  fetchTags: (filter?: string) => Promise<void>;
  assign: (resourcePath: string, tags: Record<string, string>) => Promise<boolean>;
  bulkAssign: (resourcePaths: string[], tags: Record<string, string>) => Promise<boolean>;
  remove: (id: string) => Promise<boolean>;
}

export function useResourceTags(): UseResourceTagsResult {
  const { getAccessToken } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [tags, setTags] = useState<ResourceTag[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return { 'Content-Type': 'application/json', Authorization: `Bearer ${tok}`, 'X-Tenant-ID': tenantId };
  }, [getAccessToken, tenantId]);

  const fetchTags = useCallback(async (filter?: string) => {
    const tok = getAccessToken(); if (!tok) return;
    setIsLoading(true); setError(null);
    try {
      const q = filter ? `?filter=${encodeURIComponent(filter)}` : '';
      const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-tags${q}`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed (${resp.status})`);
      setTags(await resp.json());
    } catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); }
    finally { setIsLoading(false); }
  }, [apiBaseUrl, makeHeaders]);

  const assign = useCallback(async (resourcePath: string, tagObj: Record<string, string>) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-tags`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ resource_path: resourcePath, tags: tagObj }) }); if (!resp.ok) throw new Error(`Assign failed (${resp.status})`); await fetchTags(); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchTags]);

  const bulkAssign = useCallback(async (resourcePaths: string[], tagObj: Record<string, string>) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-tags/bulk`, { method: 'POST', headers: makeHeaders(), body: JSON.stringify({ resource_paths: resourcePaths, tags: tagObj }) }); if (!resp.ok) throw new Error(`Bulk assign failed (${resp.status})`); await fetchTags(); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders, fetchTags]);

  const remove = useCallback(async (id: string) => {
    try { const resp = await fetch(`${apiBaseUrl}/api/v1/policy/resource-tags/${id}`, { method: 'DELETE', headers: makeHeaders() }); if (!resp.ok) throw new Error(`Delete failed (${resp.status})`); setTags((prev: any) => prev.filter((t) => t.id !== id)); return true; }
    catch (err) { setError(err instanceof Error ? err.message : 'Unknown'); return false; }
  }, [apiBaseUrl, makeHeaders]);

  return { tags, isLoading, error, fetchTags, assign, bulkAssign, remove };
}
