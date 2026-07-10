export interface GGIDConfig {
  /** Gateway base URL, e.g. https://iam.example.com */
  gatewayUrl: string;
  /** JWKS endpoint for JWT verification (omit for offline mode) */
  jwksUrl?: string;
  /** API key for management operations */
  apiKey?: string;
  /** Default tenant ID */
  tenantId?: string;
  /** JWT issuer for verification */
  issuer?: string;
  /** Request timeout in ms */
  timeout?: number;
}

export interface User {
  id: string;
  tenant_id?: string;
  username: string;
  email: string;
  phone?: string;
  status: string;
  email_verified?: boolean;
  display_name?: string;
  created_at?: string;
  updated_at?: string;
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
  tenant_id?: string;
  key: string;
  name: string;
  description?: string;
  system_role?: boolean;
  created_at?: string;
}

export interface Organization {
  id: string;
  tenant_id?: string;
  name: string;
  parent_id?: string;
  description?: string;
  created_at?: string;
}

export interface PolicyCheckResult {
  allowed: boolean;
  reason?: string;
}

export interface PageResult<T> {
  items: T[];
  total_count: number;
  page: number;
  page_size: number;
}

export interface ListOptions {
  page?: number;
  page_size?: number;
  search?: string;
  status?: string;
}

export interface LoginInput {
  username: string;
  password: string;
}

export interface CreateUserInput {
  username: string;
  email: string;
  password: string;
  phone?: string;
}

export interface UpdateUserInput {
  email?: string;
  phone?: string;
  status?: string;
}

export interface CreateRoleInput {
  key: string;
  name: string;
  description?: string;
}

export interface CreateOrgInput {
  name: string;
  parent_id?: string;
  description?: string;
}
