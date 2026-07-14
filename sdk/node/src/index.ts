/**
 * GGID IAM Platform Node.js SDK
 *
 * JWT verification, user management, RBAC permission checking,
 * and Express middleware for authentication and authorization.
 */

// Client + errors
export { GGIDClient, GGIDError } from './client';

// Token manager (auto-refresh)
export { TokenManager } from './token_manager';

// JWT verifier
export { JWTVerifier, JWTError } from './jwt';
export type { JWTClaims } from './jwt';

// Express middleware
export {
  expressAuth,
  requireRole,
  requirePermission,
  getClaims,
} from './middleware';
export type { GGIDRequest, MiddlewareConfig } from './middleware';

// Types
export type {
  GGIDConfig,
  User,
  TokenSet,
  Role,
  Organization,
  PolicyCheckResult,
  PageResult,
  ListOptions,
  LoginInput,
  CreateUserInput,
  UpdateUserInput,
  CreateRoleInput,
  CreateOrgInput,
  AgentType,
  Agent,
  RegisterAgentInput,
  AgentTokenResponse,
  AgentTokenClaims,
} from './types';

// RBAC/ABAC policy management
import './policy';
export type {
  PermissionCheckResult,
  ABACEvaluateRequest,
  ABACEvaluateResult,
  PolicyCheckRequest,
  Permission,
} from './policy';

// Token introspection
import './introspect';
export type { IntrospectResult } from './introspect';

// OIDC Discovery
import './discovery';
export type { DiscoveryConfig } from './discovery';
