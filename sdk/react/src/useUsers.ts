/**
 * GGID React SDK — useUsers hook
 *
 * User list + CRUD + role assignment.
 *
 * Usage:
 *   const { users, isLoading, createUser, updateUser, deleteUser, assignRole, refetch } = useUsers();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface GGIDUserRecord {
  id: string;
  username: string;
  email: string;
  tenant_id: string;
  status: 'active' | 'suspended' | 'locked' | 'pending';
  roles?: string[];
  created_at: string;
  updated_at?: string;
  last_login?: string;
  mfa_enabled?: boolean;
}

export interface CreateUserInput {
  username: string;
  email: string;
  password: string;
  roles?: string[];
}

export interface UpdateUserInput {
  email?: string;
  status?: string;
  roles?: string[];
}

export interface UseUsersResult {
  users: GGIDUserRecord[];
  isLoading: boolean;
  error: string | null;
  createUser: (input: CreateUserInput) => Promise<GGIDUserRecord | null>;
  updateUser: (id: string, input: UpdateUserInput) => Promise<boolean>;
  deleteUser: (id: string) => Promise<boolean>;
  assignRole: (userId: string, roleId: string) => Promise<boolean>;
  removeRole: (userId: string, roleId: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useUsers(): UseUsersResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [users, setUsers] = useState<GGIDUserRecord[]>([]);
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

  const fetchUsers = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/users`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch users (${resp.status})`);
      const data = await resp.json();
      setUsers(data.users ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setUsers([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchUsers();
  }, [isAuthenticated, fetchUsers]);

  const createUser = useCallback(
    async (input: CreateUserInput): Promise<GGIDUserRecord | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create user (${resp.status})`);
        const created = await resp.json();
        await fetchUsers();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchUsers]
  );

  const updateUser = useCallback(
    async (id: string, input: UpdateUserInput): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update user (${resp.status})`);
        await fetchUsers();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchUsers]
  );

  const deleteUser = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete user (${resp.status})`);
        await fetchUsers();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchUsers]
  );

  const assignRole = useCallback(
    async (userId: string, roleId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users/${userId}/roles`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify({ role_id: roleId }),
        });
        if (!resp.ok) throw new Error(`Failed to assign role (${resp.status})`);
        await fetchUsers();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchUsers]
  );

  const removeRole = useCallback(
    async (userId: string, roleId: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/users/${userId}/roles/${roleId}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to remove role (${resp.status})`);
        await fetchUsers();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchUsers]
  );

  return {
    users,
    isLoading,
    error,
    createUser,
    updateUser,
    deleteUser,
    assignRole,
    removeRole,
    refetch: fetchUsers,
  };
}
