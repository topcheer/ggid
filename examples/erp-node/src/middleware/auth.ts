/**
 * Auth middleware — JWT verification + permission checks.
 * Uses GGID Node SDK patterns for token verification.
 */
import type { Request, Response, NextFunction } from 'express';

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

// In-memory token cache (production: use Redis or JWT verification)
const tokenCache = new Map<string, ERPUser>();

/** Verify JWT token via GGID API and extract user info */
export async function verifyToken(token: string): Promise<ERPUser | null> {
  // Check cache
  if (tokenCache.has(token)) return tokenCache.get(token)!;

  try {
    const res = await fetch(`${GGID_URL}/api/v1/auth/verify`, {
      headers: { Authorization: `Bearer ${token}`, 'X-Tenant-ID': TENANT },
    });
    if (!res.ok) return null;
    const data = await res.json();

    const user: ERPUser = {
      user_id: data.user_id || data.sub || '',
      username: data.username || data.preferred_username || 'user',
      email: data.email || '',
      tenant_id: data.tenant_id || data.tenant_id || '',
      roles: data.roles || [],
      permissions: data.permissions || [],
    };

    // Cache for 5 minutes
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
    if (perms.some(p => hasPermission(user, p))) {
      return next();
    }
    return res.status(403).json({ error: { code: 'forbidden', message: `Requires any of: ${perms.join(', ')}` } });
  };
}
