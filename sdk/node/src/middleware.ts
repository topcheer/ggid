/**
 * Express/Fastify middleware for GGID JWT authentication.
 */

import type { Request, Response, NextFunction } from 'express';
import { JWTVerifier, JWTClaims, JWTError } from './jwt';
import { GGIDClient } from './client';
import type { GGIDConfig } from './types';

const DEFAULT_PUBLIC_PATHS = new Set([
  '/', '/healthz', '/docs', '/api-docs', '/login', '/register',
]);

export interface GGIDRequest extends Request {
  ggidUser?: JWTClaims;
}

export interface MiddlewareConfig {
  /** Additional path prefixes to skip authentication. */
  publicPaths?: string[];
  /** Tenant ID injected as X-Tenant-ID header. */
  tenantId?: string;
}

/**
 * Express middleware for JWT authentication.
 *
 * ```ts
 * import { expressAuth } from '@ggid/sdk';
 *
 * app.use(expressAuth(
 *   { jwksUrl: 'https://iam.example.com/.well-known/jwks.json' },
 *   { publicPaths: ['/webhooks'] },
 * ));
 * ```
 */
export function expressAuth(
  config: Pick<GGIDConfig, 'jwksUrl' | 'issuer'>,
  mwConfig?: MiddlewareConfig,
) {
  const verifier = new JWTVerifier(config);

  return async (req: GGIDRequest, res: Response, next: NextFunction) => {
    const path = req.path;

    // Check public paths (prefix match).
    if (
      DEFAULT_PUBLIC_PATHS.has(path) ||
      path.startsWith('/api/v1/auth/') ||
      path.startsWith('/oauth/') ||
      mwConfig?.publicPaths?.some((p) => path.startsWith(p))
    ) {
      return next();
    }

    const authHeader = req.headers.authorization || '';
    if (!authHeader.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'missing bearer token' });
    }

    const token = authHeader.slice(7);
    try {
      const claims = await verifier.verify(token);
      req.ggidUser = claims;

      if (mwConfig?.tenantId) {
        req.headers['x-tenant-id'] = mwConfig.tenantId;
      }

      next();
    } catch (err) {
      const msg = err instanceof JWTError ? err.message : 'invalid token';
      return res.status(401).json({ error: msg });
    }
  };
}

/**
 * Require the authenticated user to have a specific role (from JWT claims).
 */
export function requireRole(role: string) {
  return (req: GGIDRequest, res: Response, next: NextFunction) => {
    if (!req.ggidUser) {
      return res.status(401).json({ error: 'not authenticated' });
    }
    const roles = req.ggidUser.roles || [];
    if (!roles.includes(role)) {
      return res.status(403).json({ error: `requires role: ${role}` });
    }
    next();
  };
}

/**
 * Require the authenticated user to have a specific permission (via policy engine API).
 */
export function requirePermission(
  config: Pick<GGIDConfig, 'gatewayUrl' | 'tenantId' | 'apiKey'>,
  resource: string,
  action: string,
) {
  const client = new GGIDClient(config);

  return async (req: GGIDRequest, res: Response, next: NextFunction) => {
    if (!req.ggidUser) {
      return res.status(401).json({ error: 'not authenticated' });
    }

    try {
      const result = await client.checkPermission(req.ggidUser.sub, resource, action);
      if (!result.allowed) {
        return res.status(403).json({ error: 'permission denied' });
      }
      next();
    } catch {
      return res.status(500).json({ error: 'permission check failed' });
    }
  };
}

/**
 * Extract JWT claims from the request.
 */
export function getClaims(req: GGIDRequest): JWTClaims {
  if (!req.ggidUser) {
    throw new Error('not authenticated');
  }
  return req.ggidUser;
}
