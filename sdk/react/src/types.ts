/**
 * GGID React SDK — Type Definitions
 */

export interface GGIDConfig {
  /** Base URL of the GGID Gateway (e.g. https://api.ggid.dev) */
  apiBaseUrl: string;
  /** Tenant ID for multi-tenant isolation */
  tenantId: string;
 /** OAuth client ID (optional, for OIDC flow) */
  clientId?: string;
  /** Redirect URI after login */
  redirectUri?: string;
  /** Scopes to request */
  scopes?: string[];
  /** Token storage key (default: 'ggid_token') */
  storageKey?: string;
}

export interface GGIDUser {
  id: string;
  username: string;
  email: string;
  tenant_id: string;
  roles?: string[];
  scopes?: string[];
}

export interface GGIDTokenSet {
  access_token: string;
  refresh_token?: string;
  expires_at?: number;
  token_type?: string;
}

export interface GGIDAuthState {
  user: GGIDUser | null;
  tokenSet: GGIDTokenSet | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  error: string | null;
}

export interface GGIDAuthContextValue extends GGIDAuthState {
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  getAccessToken: () => string | null;
  hasRole: (role: string) => boolean;
  hasScope: (scope: string) => boolean;
}
