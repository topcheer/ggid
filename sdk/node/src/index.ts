/**
 * GGID IAM SDK — Node.js / TypeScript client.
 * Integrate GGID identity and access management into your backend.
 */

import { jwtVerify, createRemoteJWKSet } from "jose";

export interface GGIDClientOptions {
  baseURL: string;
  apiKey?: string;
}

export interface UserInfo {
  userId: string;
  tenantId: string;
  username: string;
  email: string;
  roles: string[];
  scopes: string[];
  claims: Record<string, unknown>;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  phone?: string;
}

export interface User {
  id: string;
  tenant_id: string;
  username: string;
  email: string;
  phone: string;
  status: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface TokenSet {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
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

export class GGIDClient {
  private baseURL: string;
  private apiKey?: string;
  private jwks?: ReturnType<typeof createRemoteJWKSet>;

  constructor(opts: GGIDClientOptions) {
    this.baseURL = opts.baseURL.replace(/\/$/, "");
    this.apiKey = opts.apiKey;
  }

  /** Verify a JWT access token and return user info. */
  async verifyToken(accessToken: string): Promise<UserInfo> {
    if (!this.jwks) {
      this.jwks = createRemoteJWKSet(new URL(this.baseURL + "/oauth/jwks"));
    }
    const { payload } = await jwtVerify(accessToken, this.jwks);
    return {
      userId: String(payload.sub ?? ""),
      tenantId: String(payload.tenant_id ?? ""),
      username: String(payload.username ?? ""),
      email: String(payload.email ?? ""),
      roles: Array.isArray(payload.roles) ? payload.roles.map(String) : [],
      scopes: typeof payload.scope === "string" ? payload.scope.split(" ") : [],
      claims: payload as Record<string, unknown>,
    };
  }

  /** Check if a user has permission for an action on a resource. */
  async checkPermission(userId: string, resource: string, action: string): Promise<boolean> {
    const resp = await this.post("/api/v1/policies/check", { user_id: userId, resource, action });
    return resp.allowed === true;
  }

  /** Refresh an access token. */
  async refreshToken(refreshToken: string): Promise<TokenSet> {
    return this.post("/api/v1/auth/refresh", { refresh_token: refreshToken });
  }

  /** Create a new user (requires API key). */
  async createUser(req: CreateUserRequest): Promise<User> {
    return this.post("/api/v1/users", req);
  }

  /** Get a user by ID. */
  async getUser(userId: string): Promise<User> {
    return this.get(`/api/v1/users/${userId}`);
  }

  /** List users with pagination. */
  async listUsers(opts?: ListOptions): Promise<PageResult<User>> {
    return this.get("/api/v1/users", opts);
  }

  // --- HTTP helpers ---

  private async get<T>(path: string, params?: Record<string, unknown>): Promise<T> {
    const url = new URL(this.baseURL + path);
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v !== undefined) url.searchParams.set(k, String(v));
      }
    }
    const resp = await fetch(url, { headers: this.headers() });
    return this.handleResponse(resp);
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    const resp = await fetch(this.baseURL + path, {
      method: "POST",
      headers: { ...this.headers(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    return this.handleResponse(resp);
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = {};
    if (this.apiKey) h["X-API-Key"] = this.apiKey;
    return h;
  }

  private async handleResponse<T>(resp: Response): Promise<T> {
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`GGID API error (${resp.status}): ${text}`);
    }
    return resp.json() as Promise<T>;
  }
}
