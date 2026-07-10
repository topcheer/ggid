/**
 * Express/Fastify middleware for GGID JWT authentication.
 */

import type { Request, Response, NextFunction } from 'express';
import { JWTVerifier, JWTClaims, JWTError } from './jwt';
import type { GGIDConfig } from './types';

const PUBLIC_PATHS = new Set(['/', '/healthz', '/docs', '/api-docs', '/login', '/register']);

export interface GGIDRequest extends Request {
  ggidUser?: JWTClaims;
}

export function expressAuth(config: Pick<GGIDConfig, 'jwksUrl' | 'issuer'>) {
  const verifier = new JWTVerifier(config);

  return async (req: GGIDRequest, res: Response, next: NextFunction) => {
    const path = req.path;

    // Skip public paths
    if (PUBLIC_PATHS.has(path) || path.startsWith('/api/v1/auth/') || path.startsWith('/oauth/')) {
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
      next();
    } catch (err) {
      const msg = err instanceof JWTError ? err.message : 'invalid token';
      return res.status(401).json({ error: msg });
    }
  };
}

export function requirePermission(resource: string, action: string) {
  return async (req: GGIDRequest, res: Response, next: NextFunction) => {
    // TODO: Call policy check API
    // For now, just require a valid token (expressAuth must run first)
    if (!req.ggidUser) {
      return res.status(401).json({ error: 'not authenticated' });
    }
    next();
  };
}

export function getClaims(req: GGIDRequest): JWTClaims {
  if (!req.ggidUser) {
    throw new Error('not authenticated');
  }
  return req.ggidUser;
}
