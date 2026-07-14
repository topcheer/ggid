/**
 * OIDC Discovery (RFC 8414) — GET /.well-known/openid-configuration
 */

import { GGIDClient } from './client';

export interface DiscoveryConfig {
  issuer: string;
  authorization_endpoint: string;
  token_endpoint: string;
  userinfo_endpoint?: string;
  jwks_uri: string;
  end_session_endpoint?: string;
  revocation_endpoint?: string;
  introspection_endpoint?: string;
  registration_endpoint?: string;
  device_authorization_endpoint?: string;
  scopes_supported?: string[];
  response_types_supported?: string[];
  grant_types_supported?: string[];
  subject_types_supported?: string[];
  id_token_signing_alg_values_supported?: string[];
  token_endpoint_auth_methods_supported?: string[];
  claims_supported?: string[];
  code_challenge_methods_supported?: string[];
  [key: string]: unknown;
}

declare module './client' {
  interface GGIDClient {
    getDiscovery(): Promise<DiscoveryConfig>;
  }
}

const proto = GGIDClient.prototype as any;

proto.getDiscovery = async function (): Promise<DiscoveryConfig> {
  return this.request('GET', '/.well-known/openid-configuration');
};

export {};
