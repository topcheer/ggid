/**
 * GGID React SDK — useRoles hook
 *
 * Provides role and scope helpers built on top of useGGIDAuth.
 * - roles: string[] from the current user
 * - scopes: string[] from the current user
 * - hasRole(role): checks if user has a specific role
 * - hasScope(scope): checks if user has a specific scope
 * - hasAnyRole(...roles): true if user has any of the given roles
 * - hasAllRoles(...roles): true if user has all of the given roles
 */

import { useMemo, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface UseRolesResult {
  roles: string[];
  scopes: string[];
  hasRole: (role: string) => boolean;
  hasScope: (scope: string) => boolean;
  hasAnyRole: (...roles: string[]) => boolean;
  hasAllRoles: (...roles: string[]) => boolean;
  hasAnyScope: (...scopes: string[]) => boolean;
  isAdmin: boolean;
}

export function useRoles(): UseRolesResult {
  const { user } = useGGIDAuth();

  const roles = useMemo(() => user?.roles ?? [], [user?.roles]);
  const scopes = useMemo(() => user?.scopes ?? [], [user?.scopes]);

  const hasRole = useCallback(
    (role: string) => roles.includes(role),
    [roles]
  );

  const hasScope = useCallback(
    (scope: string) => scopes.includes(scope),
    [scopes]
  );

  const hasAnyRole = useCallback(
    (...checkRoles: string[]) => checkRoles.some((r) => roles.includes(r)),
    [roles]
  );

  const hasAllRoles = useCallback(
    (...checkRoles: string[]) => checkRoles.every((r) => roles.includes(r)),
    [roles]
  );

  const hasAnyScope = useCallback(
    (...checkScopes: string[]) => checkScopes.some((s) => scopes.includes(s)),
    [scopes]
  );

  const isAdmin = useMemo(() => hasRole('admin') || hasScope('admin'), [hasRole, hasScope]);

  return {
    roles,
    scopes,
    hasRole,
    hasScope,
    hasAnyRole,
    hasAllRoles,
    hasAnyScope,
    isAdmin,
  };
}
