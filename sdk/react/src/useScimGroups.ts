/**
 * GGID React SDK — useScimGroups hook
 *
 * SCIM 2.0 group provisioning CRUD.
 *
 * Usage:
 *   const { groups, createGroup, updateGroup, deleteGroup } = useScimGroups();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ScimGroup {
  id: string;
  display_name: string;
  members: { value: string; display: string }[];
  meta: { created: string; last_modified: string };
}

export interface CreateScimGroupInput {
  display_name: string;
  member_ids?: string[];
}

export interface UseScimGroupsResult {
  groups: ScimGroup[];
  isLoading: boolean;
  error: string | null;
  createGroup: (input: CreateScimGroupInput) => Promise<ScimGroup | null>;
  updateGroup: (id: string, input: Partial<ScimGroup>) => Promise<boolean>;
  deleteGroup: (id: string) => Promise<boolean>;
  addMember: (groupId: string, userId: string) => Promise<boolean>;
  removeMember: (groupId: string, userId: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useScimGroups(): UseScimGroupsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [groups, setGroups] = useState<ScimGroup[]>([]);
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

  const fetchGroups = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch SCIM groups (${resp.status})`);
      const data = await resp.json();
      setGroups(data.Resources ?? data.groups ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setGroups([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchGroups();
  }, [isAuthenticated, fetchGroups]);

  const createGroup = useCallback(
    async (input: CreateScimGroupInput): Promise<ScimGroup | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups`, {
          method: 'POST', headers: makeHeaders(),
          body: JSON.stringify({
            displayName: input.display_name,
            members: (input.member_ids ?? []).map((id) => ({ value: id })),
          }),
        });
        if (!resp.ok) throw new Error(`Failed to create group (${resp.status})`);
        const created = await resp.json();
        await fetchGroups();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchGroups],
  );

  const updateGroup = useCallback(
    async (id: string, input: Partial<ScimGroup>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups/${id}`, {
          method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update group (${resp.status})`);
        await fetchGroups();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchGroups],
  );

  const deleteGroup = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups/${id}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete group (${resp.status})`);
        setGroups((prev) => prev.filter((g: any) => g.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const addMember = useCallback(
    async (groupId: string, userId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups/${groupId}/members`, {
          method: 'POST', headers: makeHeaders(), body: JSON.stringify({ value: userId }),
        });
        if (!resp.ok) throw new Error(`Failed to add member (${resp.status})`);
        await fetchGroups();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchGroups],
  );

  const removeMember = useCallback(
    async (groupId: string, userId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/scim/Groups/${groupId}/members/${userId}`, {
          method: 'DELETE', headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to remove member (${resp.status})`);
        await fetchGroups();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchGroups],
  );

  return {
    groups, isLoading, error,
    createGroup, updateGroup, deleteGroup, addMember, removeMember,
    refetch: fetchGroups,
  };
}
