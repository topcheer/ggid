/**
 * Auth middleware — JWT verification via JWKS + permission checks.
 * Fetches JWKS manually and uses jsonwebtoken for verification.
 */
import type { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import crypto from 'crypto';

const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const TENANT = process.env.GGID_TENANT || '00000000-0000-0000-0000-000000000002';

export interface ERPUser {
  user_id: string;
  username: string;
  email: string;
  tenant_id: string;
  roles: string[];
  permissions: string[];
}

// JWKS cache
let jwksCache: any = null;
let jwksCachedAt = 0;

async function getSigningKey(kid: string): Promise<string> {
  const now = Date.now();
  if (!jwksCache || now - jwksCachedAt > 300000) {
    const resp = await fetch(`${GGID_URL}/.well-known/jwks.json`);
    jwksCache = await resp.json();
    jwksCachedAt = now;
  }

  const key = jwksCache.keys?.find((k: any) => k.kid === kid);
  if (!key) throw new Error(`No matching key for kid: ${kid}`);

  // Convert JWK to PEM
  const pubKey = crypto.createPublicKey({
    key: {
      kty: key.kty,
      n: key.n,
      e: key.e,
    },
    format: 'jwk',
  });
  return pubKey.export({ type: 'spki', format: 'pem' }) as string;
}

// In-memory token cache
const tokenCache = new Map<string, ERPUser>();

/** Verify JWT token via JWKS and extract user info */
export async function verifyToken(token: string): Promise<ERPUser | null> {
  if (tokenCache.has(token)) return tokenCache.get(token)!;

  try {
    // Decode header to get kid
    const decoded = jwt.decode(token, { complete: true });
    if (!decoded || typeof decoded === 'string') return null;
    const kid = decoded.header.kid;
    if (!kid) return null;

    const signingKey = await getSigningKey(kid);
    const payload: any = jwt.verify(token, signingKey, { algorithms: ['RS256'] });

    const user: ERPUser = {
      user_id: payload.sub || '',
      username: payload.username || payload.preferred_username || payload.sub || 'user',
      email: payload.email || '',
      tenant_id: payload.tenant_id || '',
      roles: payload.roles || [],
      permissions: payload.permissions || [],
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
  return user.permissions.includes(perm);
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
