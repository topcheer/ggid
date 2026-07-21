/**
 * GGID React SDK — RequireScope component
 *
 * Conditionally renders children only when the user has the required scope(s).
 *
 * Usage:
 * // Single scope
 * <RequireScope scope="admin">
 *   <AdminPanel />
 * </RequireScope>
 *
 * // Any of multiple scopes
 * <RequireScope anyOf={['admin', 'user-manager']}>
 *   <ManageUsers />
 * </RequireScope>
 *
 * // All of multiple scopes
 * <RequireScope allOf={['users:read', 'users:write']}>
 *   <EditUsers />
 * </RequireScope>
 *
 * // With fallback
 * <RequireScope scope="admin" fallback={<AccessDenied />}>
 *   <AdminPanel />
 * </RequireScope>
 */

import React from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RequireScopeProps {
  /** Single required scope */
  scope?: string;
  /** User must have ANY of these scopes */
  anyOf?: string[];
  /** User must have ALL of these scopes */
  allOf?: string[];
  /** Fallback content when scope check fails (default: null) */
  fallback?: React.ReactNode;
  /** Content to show while loading auth state */
  loadingFallback?: React.ReactNode;
  /** Children to render if authorized */
  children: React.ReactNode;
}

export function RequireScope({
  scope,
  anyOf,
  allOf,
  fallback = null,
  loadingFallback = null,
  children,
}: RequireScopeProps) {
  const { hasScope, isLoading } = useGGIDAuth();

  if (isLoading) {
    return <>{loadingFallback}</>;
  }

  let authorized = true;

  if (scope) {
    authorized = authorized && hasScope(scope);
  }

  if (anyOf && anyOf.length > 0) {
    authorized = authorized && anyOf.some((s) => hasScope(s));
  }

  if (allOf && allOf.length > 0) {
    authorized = authorized && allOf.every((s) => hasScope(s));
  }

  if (!authorized) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
}
