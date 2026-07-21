/**
 * JWT verification using jose library with JWKS caching.
 */

import { jwtVerify, createRemoteJWKSet } from 'jose';
import type { GGIDConfig } from './types';

export interface JWTClaims {
  sub: string;
  email?: string;
  name?: string;
  tenant_id?: string;
  roles?: string[];
  permissions?: string[];
  aud?: string | string[];
  exp?: number;
  iat?: number;
  iss?: string;
  [key: string]: unknown;
}

export class JWTError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'JWTError';
  }
}

export class JWTVerifier {
  private jwks: ReturnType<typeof createRemoteJWKSet>;
  private issuer?: string;

  constructor(config: Pick<GGIDConfig, 'jwksUrl' | 'issuer'>) {
    if (!config.jwksUrl) {
      throw new Error('jwksUrl is required for JWTVerifier');
    }
    this.jwks = createRemoteJWKSet(new URL(config.jwksUrl));
    this.issuer = config.issuer;
  }

  async verify(token: string): Promise<JWTClaims> {
    try {
      const { payload } = await jwtVerify(token, this.jwks, {
        algorithms: ['RS256'],
        issuer: this.issuer,
      });
      return payload as unknown as JWTClaims;
    } catch (err: any) {
      if (err.code === 'ERR_JWT_EXPIRED') {
        throw new JWTError('token expired');
      }
      throw new JWTError(`invalid token: ${err.message}`);
    }
  }
}
