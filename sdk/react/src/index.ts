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
export { useUsers } from './useUsers';
export type { GGIDUserRecord, CreateUserInput, UpdateUserInput, UseUsersResult } from './useUsers';
export { useBranding } from './useBranding';
export type { BrandingConfig, UseBrandingResult } from './useBranding';
export { useRetention } from './useRetention';
export type { RetentionPolicy, UseRetentionResult } from './useRetention';
export { useAlerts } from './useAlerts';
export type { AlertRule, CreateAlertRuleInput, UseAlertsResult } from './useAlerts';
export { useSessions } from './useSessions';
export type { UserSession, UseSessionsResult } from './useSessions';
export { useCompliance } from './useCompliance';
export type { ComplianceFramework, ComplianceReport, ComplianceFilter, UseComplianceResult } from './useCompliance';
export { useOAuthClients } from './useOAuthClients';
export type { OAuthClient, CreateOAuthClientInput, UseOAuthClientsResult } from './useOAuthClients';
export { useAuditStats } from './useAuditStats';
export type { AuditStats, HourlyBucket, TopActor, UseAuditStatsResult } from './useAuditStats';
export { useOrgs } from './useOrgs';
export type { Organization, CreateOrgInput, UpdateOrgInput, UseOrgsResult } from './useOrgs';
export { usePolicies } from './usePolicies';
export type { Policy, ABACRule, CreatePolicyInput, UpdatePolicyInput, UsePoliciesResult } from './usePolicies';
export type {
  GGIDConfig,
  GGIDUser,
  GGIDTokenSet,
  GGIDAuthState,
  GGIDAuthContextValue,
} from './types';
