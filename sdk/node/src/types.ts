export interface GGIDConfig {
  /** Gateway base URL, e.g. https://iam.example.com */
  gatewayUrl: string;
  /** JWKS endpoint for JWT verification */
  jwksUrl: string;
  /** Default tenant ID */
  tenantId?: string;
  /** JWT issuer for verification */
  issuer?: string;
  /** Request timeout in ms */
  timeout?: number;
}

export interface User {
  id: string;
  username: string;
  email: string;
  status: string;
  display_name?: string;
  created_at?: string;
}

export interface TokenSet {
  access_token: string;
  refresh_token?: string;
  id_token?: string;
  token_type: string;
  expires_in: number;
}

export interface Role {
  id: string;
  name: string;
  key: string;
  description?: string;
  system_role: boolean;
}

export interface PolicyCheckResult {
  allowed: boolean;
  reason?: string;
}
