/**
 * GGID React SDK — useGroups hook
 *
 * Group CRUD + member add/remove.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Group {
  id: string;
  name: string;
  description: string;
  member_count: number;
  parent_id?: string;
  created_at: string;
  updated_at?: string;
}

export interface GroupMember {
  id: string;
  user_id: string;
  username: string;
  email: string;
  role: 'member' | 'manager' | 'owner';
  added_at: string;
}

export interface CreateGroupInput {
  name: string;
  description?: string;
  parent_id?: string;
}

export interface UseGroupsResult {
  groups: Group[];
  isLoading: boolean;
  error: string | null;
  createGroup: (input: CreateGroupInput) => Promise<Group | null>;
  updateGroup: (id: string, input: Partial<Group>) => Promise<boolean>;
  deleteGroup: (id: string) => Promise<boolean>;
  getMembers: (groupId: string) => Promise<GroupMember[]>;
  addMember: (groupId: string, userId: string, role?: string) => Promise<boolean>;
  removeMember: (groupId: string, userId: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useGroups(): UseGroupsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [groups, setGroups] = useState<Group[]>([]);
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
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch groups (${resp.status})`);
      const data = await resp.json();
      setGroups(data.groups ?? data.items ?? []);
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

  const createGroup = useCallback(async (input: CreateGroupInput): Promise<Group | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to create group (${resp.status})`);
      const created = await resp.json();
      await fetchGroups();
      return created;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders, fetchGroups]);

  const updateGroup = useCallback(async (id: string, input: Partial<Group>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups/${id}`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to update group (${resp.status})`);
      await fetchGroups();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchGroups]);

  const deleteGroup = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to delete group (${resp.status})`);
      await fetchGroups();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchGroups]);

  const getMembers = useCallback(async (groupId: string): Promise<GroupMember[]> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups/${groupId}/members`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch members (${resp.status})`);
      const data = await resp.json();
      return data.members ?? data.items ?? [];
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    }
  }, [apiBaseUrl, makeHeaders]);

  const addMember = useCallback(async (groupId: string, userId: string, role = 'member'): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups/${groupId}/members`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify({ user_id: userId, role }),
      });
      if (!resp.ok) throw new Error(`Failed to add member (${resp.status})`);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  const removeMember = useCallback(async (groupId: string, userId: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/groups/${groupId}/members/${userId}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to remove member (${resp.status})`);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders]);

  return {
    groups, isLoading, error,
    createGroup, updateGroup, deleteGroup,
    getMembers, addMember, removeMember,
    refetch: fetchGroups,
  };
}
