/**
 * JWT verification using Node.js built-in crypto (no jose dependency).
 * Verifies RS256 JWT tokens against GGID JWKS endpoint.
 */

import { createPublicKey, verify as cryptoVerify, createHash } from 'crypto';
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
  private jwksUrl: string;
  private gatewayUrl?: string;
  private issuer?: string;
  private cachedKeys: Map<string, string> = new Map(); // kid -> PEM public key
  private cacheExpiry = 0;

  constructor(config: Pick<GGIDConfig, 'jwksUrl' | 'issuer' | 'gatewayUrl'>) {
    // Auto-derive jwksUrl from gatewayUrl if not explicitly provided
    if (config.jwksUrl) {
      this.jwksUrl = config.jwksUrl;
    } else if (config.gatewayUrl) {
      this.jwksUrl = `${config.gatewayUrl}/.well-known/jwks.json`;
    } else {
      throw new Error('Either jwksUrl or gatewayUrl is required');
    }
    this.issuer = config.issuer;
  }

  private async getKeyPem(kid: string): Promise<string> {
    if (this.cachedKeys.has(kid) && Date.now() < this.cacheExpiry) {
      return this.cachedKeys.get(kid)!;
    }

    const resp = await fetch(this.jwksUrl, {
      headers: { 'Accept-Encoding': 'identity' },
    });
    if (!resp.ok) throw new JWTError(`JWKS fetch failed: ${resp.status}`);
    const jwks = await resp.json() as { keys: any[] };

    for (const key of jwks.keys || []) {
      if (key.kty === 'RSA' && key.n && key.e) {
        const pem = jwkToPem(key.n, key.e);
        this.cachedKeys.set(key.kid, pem);
      }
    }
    this.cacheExpiry = Date.now() + 300_000; // 5 min cache

    const pem = this.cachedKeys.get(kid);
    if (!pem) throw new JWTError(`key not found for kid: ${kid}`);
    return pem;
  }

  async verify(token: string): Promise<JWTClaims> {
    const parts = token.split('.');
    if (parts.length !== 3) throw new JWTError('invalid token format');

    const header = JSON.parse(Buffer.from(parts[0], 'base64url').toString()) as { kid?: string; alg?: string };
    if (header.alg !== 'RS256') throw new JWTError(`unsupported alg: ${header.alg}`);
    if (!header.kid) throw new JWTError('missing kid in token header');

    const pem = await this.getKeyPem(header.kid);

    const verified = cryptoVerify('RSA-SHA256', Buffer.from(`${parts[0]}.${parts[1]}`), pem, Buffer.from(parts[2], 'base64url'));
    if (!verified) throw new JWTError('signature verification failed');

    const claims = JSON.parse(Buffer.from(parts[1], 'base64url').toString()) as JWTClaims;

    if (this.issuer && claims.iss !== this.issuer) {
      throw new JWTError(`issuer mismatch: expected ${this.issuer}, got ${claims.iss}`);
    }

    if (claims.exp && Date.now() / 1000 > claims.exp) {
      throw new JWTError('token expired');
    }

    return claims;
  }
}

/** Convert JWK n+e to PEM public key using Node.js crypto */
function jwkToPem(nB64: string, eB64: string): string {
  const n = Buffer.from(nB64, 'base64url');
  const e = Buffer.from(eB64, 'base64url');
  const keyObject = createPublicKey({
    key: { kty: 'RSA', n: n.toString('base64url'), e: e.toString('base64url') },
    format: 'jwk',
  });
  return keyObject.export({ type: 'spki', format: 'pem' }).toString();
}
