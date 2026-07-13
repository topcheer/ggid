/**
 * RBAC/ABAC policy management methods for GGIDClient.
 * Uses prototype augmentation to add methods to GGIDClient.
 */

// Type definitions for policy methods
export interface PermissionCheckResult {
  allowed: boolean;
  reason: string;
  matched_by?: string;
}

export interface ABACEvaluateRequest {
  action: string;
  resource: string;
  subject: string;
  conditions?: Record<string, unknown>;
  tenant_id?: string;
}

export interface ABACEvaluateResult {
  allowed: boolean;
  reason: string;
  matched_rules?: string[];
  decision?: string;
}

export interface PolicyCheckRequest {
  subject: string;
  resource: string;
  action: string;
  context?: Record<string, string>;
}

export interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  description?: string;
  children?: Permission[];
}

// Import the client class to augment it
import { GGIDClient } from './client';

// Augment the GGIDClient class with RBAC/ABAC methods that are NOT already in client.ts
declare module './client' {
  interface GGIDClient {
    revokeRole(accessToken: string, userId: string, roleId: string): Promise<void>;
    getUserRoles(accessToken: string, userId: string): Promise<unknown[]>;
    listPermissions(accessToken: string): Promise<Permission[]>;
    deleteRole(accessToken: string, roleId: string): Promise<void>;
    getRole(accessToken: string, roleId: string): Promise<unknown>;
  }
}

// Implement methods via prototype. Using `any` cast to access private requestWithToken.
const proto = GGIDClient.prototype as any;

proto.revokeRole = async function (
  accessToken: string,
  userId: string,
  roleId: string,
): Promise<void> {
  await this.requestWithToken('DELETE', `/api/v1/policies/roles/${roleId}/users/${userId}`,
    undefined, accessToken);
};

proto.getUserRoles = async function (
  accessToken: string,
  userId: string,
): Promise<unknown[]> {
  return this.requestWithToken('GET', `/api/v1/policies/users/${userId}/roles`,
    undefined, accessToken);
};

proto.listPermissions = async function (
  accessToken: string,
): Promise<Permission[]> {
  return this.requestWithToken('GET', '/api/v1/policies/permissions/tree',
    undefined, accessToken);
};

proto.deleteRole = async function (
  accessToken: string,
  roleId: string,
): Promise<void> {
  await this.requestWithToken('DELETE', `/api/v1/roles/${roleId}`, undefined, accessToken);
};

proto.getRole = async function (
  accessToken: string,
  roleId: string,
): Promise<unknown> {
  return this.requestWithToken('GET', `/api/v1/roles/${roleId}`, undefined, accessToken);
};

export {};
