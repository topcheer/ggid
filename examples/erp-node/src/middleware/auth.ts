/**
 * Auth middleware — uses GGID Node SDK for JWT verification + permission checks.
 */
import type { Request, Response, NextFunction } from 'express';
import { GGIDClient, type JWTClaims } from '../../../sdk/node/src/client';

const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const TENANT = process.env.GGID_TENANT || '00000000-0000-0000-0000-000000000002';

// Create GGID SDK client with JWKS for JWT verification
const ggidClient = new GGIDClient({
  gatewayUrl: GGID_URL,
  tenantId: TENANT,
  jwksUrl: `${GGID_URL}/.well-known/jwks.json`,
});

export interface ERPUser {
  user_id: string;
  username: string;
  email: string;
  tenant_id: string;
  roles: string[];
  permissions: string[];
}

// In-memory token cache (5 min TTL)
const tokenCache = new Map<string, ERPUser>();

/** Verify JWT token using GGID SDK's verifyToken (JWKS + RS256) */
export async function verifyToken(token: string): Promise<ERPUser | null> {
  if (tokenCache.has(token)) return tokenCache.get(token)!;

  try {
    // Use SDK's verifyToken — does JWKS fetch + RS256 verification internally
    const claims: JWTClaims = await ggidClient.verifyToken(token);

    const user: ERPUser = {
      user_id: claims.sub || '',
      username: (claims as any).username || (claims as any).preferred_username || claims.sub || 'user',
      email: claims.email || '',
      tenant_id: claims.tenant_id || TENANT,
      roles: claims.roles || [],
      permissions: claims.permissions || [],
    };

    tokenCache.set(token, user);
    setTimeout(() => tokenCache.delete(token), 300000);
    return user;
  } catch {
    return null;
  }
}

/** Check if user has a specific permission */
export function hasPermission(user: ERPUser, perm: string): boolean {
  if (user.permissions.includes('admin')) return true;
  return user.permissions.includes(perm) || user.permissions.includes(`${perm}:all`);
}

/** Require authentication middleware */
export function requireAuth() {
  return async (req: Request, res: Response, next: NextFunction) => {
    const token = req.headers.authorization?.replace(/^Bearer\s+/i, '');
    if (!token) return res.status(401).json({ error: { code: 'unauthenticated', message: 'Missing token' } });

    const user = await verifyToken(token);
    if (!user) return res.status(401).json({ error: { code: 'unauthenticated', message: 'Invalid token' } });

    (req as any).user = user;
    next();
  };
}

/** Require a specific permission */
export function requirePermission(perm: string) {
  return (req: Request, res: Response, next: NextFunction) => {
    const user = (req as any).user as ERPUser;
    if (!hasPermission(user, perm)) {
      return res.status(403).json({ error: { code: 'forbidden', message: `Requires permission: ${perm}` } });
    }
    next();
  };
}

/** Require any of the given permissions */
export function requireAny(...perms: string[]) {
  return (req: Request, res: Response, next: NextFunction) => {
    const user = (req as any).user as ERPUser;
    if (perms.some(p => hasPermission(user, p))) return next();
    return res.status(403).json({ error: { code: 'forbidden', message: `Requires any of: ${perms.join(', ')}` } });
  };
}

/** Export the GGID client for use in routes (clientCredentials etc.) */
export { ggidClient };