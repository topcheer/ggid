/**
 * Token introspection (RFC 7662) — query token status.
 */

import { GGIDClient } from './client';

export interface IntrospectResult {
  active: boolean;
  scope?: string;
  client_id?: string;
  username?: string;
  token_type?: string;
  exp?: number;
  iat?: number;
  sub?: string;
  aud?: string[];
  iss?: string;
  tenant_id?: string;
  email?: string;
  roles?: string[];
  [key: string]: unknown;
}

// Augment GGIDClient with introspect method
declare module './client' {
  interface GGIDClient {
    introspect(accessToken: string, token: string): Promise<IntrospectResult>;
  }
}

const proto = GGIDClient.prototype as any;

proto.introspect = async function (
  accessToken: string,
  token: string,
): Promise<IntrospectResult> {
  return this.requestWithToken(
    'POST',
    '/api/v1/oauth/introspect',
    { token, token_type_hint: 'access_token' },
    accessToken,
  );
};

export {};
