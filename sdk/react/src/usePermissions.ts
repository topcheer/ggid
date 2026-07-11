/**
 * GGID React SDK — usePermissions hook
 *
 * Fine-grained permission checking built on top of useGGIDAuth.
 * - permissions: string[] (derived from scopes)
 * - hasPermission(perm): checks if user has a specific permission
 * - hasAnyPermission(...perms): true if user has any of the given permissions
 * - hasAllPermissions(...perms): true if user has all of the given permissions
 *
 * Permissions follow the format: resource:action (e.g. 'users:read', 'roles:write')
 * They are derived from the user's scopes, split on ':' to normalize.
 */

import { useMemo, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface UsePermissionsResult {
  permissions: string[];
  hasPermission: (permission: string) => boolean;
  hasAnyPermission: (...permissions: string[]) => boolean;
  hasAllPermissions: (...permissions: string[]) => boolean;
  loading: boolean;
}

/**
 * Normalizes a scope into a permission string.
 * Handles common scope formats:
 * - 'users:read' → 'users:read'
 * - 'user.read' → 'user:read'
 * - 'read:users' → 'users:read' (OAuth-style reversed)
 */
function normalizePermission(scope: string): string {
  // Already in resource:action format
  if (scope.includes(':')) {
    return scope;
  }
  // Dot notation: user.read → user:read
  if (scope.includes('.')) {
    return scope.replace('.', ':');
  }
  return scope;
}

/**
 * Checks if a permission is satisfied by any of the user's permissions,
 * including wildcard matching (e.g. 'users:*' satisfies 'users:read').
 */
function permissionMatches(userPerm: string, requiredPerm: string): boolean {
  // Exact match
  if (userPerm === requiredPerm) return true;

  // Wildcard match: 'users:*' matches 'users:read'
  const [userResource, userAction] = userPerm.split(':');
  const [reqResource, reqAction] = requiredPerm.split(':');

  if (userResource === reqResource) {
    if (userAction === '*') return true;
    if (userAction === 'admin') return true;
  }

  // Global admin
  if (userPerm === '*' || userPerm === 'admin' || userPerm === 'admin:*') {
    return true;
  }

  return false;
}

export function usePermissions(): UsePermissionsResult {
  const { user, isLoading } = useGGIDAuth();

  const permissions = useMemo(() => {
    const scopes = user?.scopes ?? [];
    return scopes.map(normalizePermission);
  }, [user?.scopes]);

  const hasPermission = useCallback(
    (permission: string) => {
      const normalized = normalizePermission(permission);
      return permissions.some((p) => permissionMatches(p, normalized));
    },
    [permissions]
  );

  const hasAnyPermission = useCallback(
    (...checkPerms: string[]) => checkPerms.some((p) => hasPermission(p)),
    [hasPermission]
  );

  const hasAllPermissions = useCallback(
    (...checkPerms: string[]) => checkPerms.every((p) => hasPermission(p)),
    [hasPermission]
  );

  return {
    permissions,
    hasPermission,
    hasAnyPermission,
    hasAllPermissions,
    loading: isLoading,
  };
}
