export interface GGIDConfig {
  baseUrl: string;
  tenantId: string;
  clientId: string;
  redirectUri: string;
  scopes?: string;
}

export interface GGIDUser {
  sub: string;
  name?: string;
  email?: string;
  roles: string[];
  permissions: string[];  // Fine-grained permissions
  picture?: string;
}

export interface GGIDSession {
  accessToken: string;
  refreshToken?: string;
  expiresAt: number;
  user: GGIDUser | null;
}

export interface GGIDClaims {
  sub: string;
  tenant_id: string;
  roles: string[];
  permissions: string[];  // Fine-grained permissions
  scope: string;          // OAuth scopes only
  exp: number;
  iat: number;
  iss: string;
}
