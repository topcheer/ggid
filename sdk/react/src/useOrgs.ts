/**
 * GGID React SDK — useOrgs hook
 *
 * Organization CRUD + tree endpoint.
 *
 * Usage:
 *   const { orgs, isLoading, createOrg, updateOrg, deleteOrg, getOrgTree, refetch } = useOrgs();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Organization {
  id: string;
  name: string;
  description: string;
  parent_id?: string;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at?: string;
  children?: Organization[];
}

export interface CreateOrgInput {
  name: string;
  description?: string;
  parent_id?: string;
}

export interface UpdateOrgInput {
  name?: string;
  description?: string;
  status?: string;
  parent_id?: string;
}

export interface UseOrgsResult {
  orgs: Organization[];
  isLoading: boolean;
  error: string | null;
  createOrg: (input: CreateOrgInput) => Promise<Organization | null>;
  updateOrg: (id: string, input: UpdateOrgInput) => Promise<boolean>;
  deleteOrg: (id: string) => Promise<boolean>;
  getOrgTree: () => Promise<Organization[]>;
  refetch: () => Promise<void>;
}

export function useOrgs(): UseOrgsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchOrgs = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch organizations (${resp.status})`);
      const data = await resp.json();
      setOrgs(data.orgs ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setOrgs([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchOrgs();
  }, [isAuthenticated, fetchOrgs]);

  const createOrg = useCallback(
    async (input: CreateOrgInput): Promise<Organization | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/orgs`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create organization (${resp.status})`);
        const created = await resp.json();
        await fetchOrgs();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchOrgs]
  );

  const updateOrg = useCallback(
    async (id: string, input: UpdateOrgInput): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update organization (${resp.status})`);
        await fetchOrgs();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchOrgs]
  );

  const deleteOrg = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete organization (${resp.status})`);
        await fetchOrgs();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchOrgs]
  );

  const getOrgTree = useCallback(async (): Promise<Organization[]> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/orgs/tree`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch org tree (${resp.status})`);
      const data = await resp.json();
      return data.orgs ?? data.tree ?? [];
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    orgs,
    isLoading,
    error,
    createOrg,
    updateOrg,
    deleteOrg,
    getOrgTree,
    refetch: fetchOrgs,
  };
}
