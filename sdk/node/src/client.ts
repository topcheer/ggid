/**
 * GGID API client for user management, auth, and RBAC.
 */

import type { GGIDConfig, User, TokenSet, Role, PolicyCheckResult } from './types';
import { JWTVerifier, JWTClaims, JWTError } from './jwt';

export class GGIDClient {
  private config: Required<Pick<GGIDConfig, 'gatewayUrl' | 'tenantId' | 'timeout'>>;
  private verifier?: JWTVerifier;

  constructor(config: GGIDConfig) {
    this.config = {
      gatewayUrl: config.gatewayUrl.replace(/\/$/, ''),
      tenantId: config.tenantId || '00000000-0000-0000-0000-000000000001',
      timeout: config.timeout || 30000,
    };
    if (config.jwksUrl) {
      this.verifier = new JWTVerifier({
        jwksUrl: config.jwksUrl,
        issuer: config.issuer,
      });
    }
  }

  private headers(token?: string): Record<string, string> {
    const h: Record<string, string> = {
      'X-Tenant-ID': this.config.tenantId,
      'Content-Type': 'application/json',
    };
    if (token) h['Authorization'] = `Bearer ${token}`;
    return h;
  }

  private async request<T>(method: string, path: string, body?: any, token?: string): Promise<T> {
    const resp = await fetch(`${this.config.gatewayUrl}${path}`, {
      method,
      headers: this.headers(token),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!resp.ok) {
      const text = await resp.text().catch(() => '');
      throw new Error(`GGID API ${resp.status}: ${text}`);
    }
    return resp.status === 204 ? (undefined as T) : await resp.json() as T;
  }

  // --- Auth ---

  async login(username: string, password: string): Promise<TokenSet> {
    return this.request('POST', '/api/v1/auth/login', { username, password });
  }

  async register(username: string, email: string, password: string, name?: string): Promise<{ user_id: string }> {
    return this.request('POST', '/api/v1/auth/register', { username, email, password, name });
  }

  // --- Users ---

  async listUsers(token: string, limit = 50): Promise<{ users: User[] }> {
    return this.request('GET', `/api/v1/users?limit=${limit}`, undefined, token);
  }

  async getUser(token: string, userId: string): Promise<User> {
    return this.request('GET', `/api/v1/users/${userId}`, undefined, token);
  }

  async deleteUser(token: string, userId: string): Promise<void> {
    return this.request('DELETE', `/api/v1/users/${userId}`, undefined, token);
  }

  // --- RBAC ---

  async listRoles(token: string): Promise<{ roles: Role[] }> {
    return this.request('GET', '/api/v1/roles', undefined, token);
  }

  async checkPermission(token: string, resource: string, action: string, userId?: string): Promise<PolicyCheckResult> {
    return this.request('POST', '/api/v1/policies/check', { resource, action, user_id: userId }, token);
  }

  // --- JWT ---

  async verifyToken(token: string): Promise<JWTClaims> {
    if (!this.verifier) throw new Error('no jwksUrl configured');
    return this.verifier.verify(token);
  }
}
