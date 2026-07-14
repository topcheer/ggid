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
  scope: string;
  exp: number;
  iat: number;
  iss: string;
}
