/**
 * GGID React SDK — usePermissions hook
 *
 * Fine-grained permission checking built on top of useGGIDAuth.
 * - permissions: string[] (from JWT `permissions` claim)
 * - hasPermission(perm): checks if user has a specific permission
 * - hasAnyPermission(...perms): true if user has any of the given permissions
 * - hasAllPermissions(...perms): true if user has all of the given permissions
 *
 * Permissions follow the format: resource:action (e.g. 'users:read', 'roles:write')
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
    return user?.permissions ?? [];
  }, [user?.permissions]);

  const hasPermission = useCallback(
    (permission: string) => {
      const normalized = normalizePermission(permission);
      return permissions.some((p: any) => permissionMatches(p, normalized));
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
