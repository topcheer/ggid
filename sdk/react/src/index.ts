/**
 * GGID React SDK — Entry Point
 *
 * Usage:
 *   import { GGIDProvider, useGGIDAuth, ProtectedRoute, useUser, ErrorBoundary } from '@ggid/react';
 */

export { GGIDProvider, GGIDAuthContext } from './GGIDProvider';
export { useGGIDAuth } from './useGGIDAuth';
export { useUser } from './useUser';
export { ProtectedRoute } from './ProtectedRoute';
export { ErrorBoundary } from './ErrorBoundary';
export { useTokenRefresh } from './useTokenRefresh';
export { useRoles } from './useRoles';
export type { UseRolesResult } from './useRoles';
export { usePermissions } from './usePermissions';
export type { UsePermissionsResult } from './usePermissions';
export { LogoutButton } from './LogoutButton';
export type { LogoutButtonProps } from './LogoutButton';
export { RequireScope } from './RequireScope';
export type { RequireScopeProps } from './RequireScope';
export { useAuditEvents } from './useAuditEvents';
export type { AuditEvent, AuditEventFilter, UseAuditEventsResult } from './useAuditEvents';
export { useAccessRequests } from './useAccessRequests';
export type { AccessRequest, CreateAccessRequestInput, UseAccessRequestsResult } from './useAccessRequests';
export type {
  GGIDConfig,
  GGIDUser,
  GGIDTokenSet,
  GGIDAuthState,
  GGIDAuthContextValue,
} from './types';
