/**
 * GGID IAM Platform Node.js SDK
 *
 * JWT verification, user management, and RBAC permission checking
 * for Express, Fastify, Next.js, and other Node.js frameworks.
 */

export { GGIDClient } from './client';
export { JWTVerifier, JWTClaims, JWTError } from './jwt';
export {
  expressAuth,
  requirePermission,
  getClaims,
} from './middleware';
export { GGIDConfig } from './types';
