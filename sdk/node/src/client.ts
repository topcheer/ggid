/**
 * GGID API client for user management, auth, RBAC, and organizations.
 */

import type {
  GGIDConfig,
  User,
  TokenSet,
  Role,
  Organization,
  PolicyCheckResult,
  PageResult,
  ListOptions,
  LoginInput,
  CreateUserInput,
  UpdateUserInput,
  CreateRoleInput,
  CreateOrgInput,
  Agent,
  RegisterAgentInput,
  AgentTokenResponse,
  AgentTokenClaims,
} from './types';
import { JWTVerifier, JWTClaims } from './jwt';

/**
 * Structured API error returned for all non-2xx responses.
 */
export class GGIDError extends Error {
  readonly statusCode: number;
  readonly code?: string;

  constructor(statusCode: number, message: string, code?: string) {
    super(message);
    this.name = 'GGIDError';
    this.statusCode = statusCode;
    this.code = code;
  }

  get isNotFound() { return this.statusCode === 404; }
  get isUnauthorized() { return this.statusCode === 401; }
  get isForbidden() { return this.statusCode === 403; }
  get isConflict() { return this.statusCode === 409; }
  get isRateLimited() { return this.statusCode === 429; }
}

export class GGIDClient {
  private config: Required<Pick<GGIDConfig, 'gatewayUrl' | 'tenantId' | 'timeout'>>;
  private apiKey?: string;
  private verifier?: JWTVerifier;

  constructor(config: GGIDConfig) {
    this.config = {
      gatewayUrl: config.gatewayUrl.replace(/\/$/, ''),
      tenantId: config.tenantId || '00000000-0000-0000-0000-000000000001',
      timeout: config.timeout || 30000,
    };
    this.apiKey = config.apiKey;
    if (config.jwksUrl) {
      this.verifier = new JWTVerifier({
        jwksUrl: config.jwksUrl,
        issuer: config.issuer,
      });
    }
  }

  // ---------------------------------------------------------------------------
  // Auth
  // ---------------------------------------------------------------------------

  /** Authenticate with username/password. */
  async login(input: LoginInput): Promise<TokenSet> {
    return this.request<TokenSet>('POST', '/api/v1/auth/login', input);
  }

  /** Register a new user. */
  async register(username: string, email: string, password: string): Promise<{ user_id: string }> {
    return this.request('POST', '/api/v1/auth/register', { username, email, password });
  }

  /** Logout — invalidate the given access token. */
  async logout(accessToken: string): Promise<void> {
    return this.request('POST', '/api/v1/auth/logout', { access_token: accessToken });
  }

  /** Refresh an access token using a refresh token. */
  async refreshToken(refreshToken: string): Promise<TokenSet> {
    return this.request<TokenSet>('POST', '/api/v1/auth/refresh', { refresh_token: refreshToken });
  }

  /** Verify a JWT and return claims (requires jwksUrl). */
  async verifyToken(token: string): Promise<JWTClaims> {
    if (!this.verifier) throw new Error('no jwksUrl configured');
    return this.verifier.verify(token);
  }

  // ---------------------------------------------------------------------------
  // Users
  // ---------------------------------------------------------------------------

  async createUser(input: CreateUserInput): Promise<User> {
    return this.request<User>('POST', '/api/v1/users', input);
  }

  async getUser(userId: string): Promise<User> {
    return this.request<User>('GET', `/api/v1/users/${userId}`);
  }

  async updateUser(userId: string, input: UpdateUserInput): Promise<User> {
    return this.request<User>('PATCH', `/api/v1/users/${userId}`, input);
  }

  async deleteUser(userId: string): Promise<void> {
    return this.request('DELETE', `/api/v1/users/${userId}`);
  }

  async listUsers(opts?: ListOptions): Promise<PageResult<User>> {
    return this.request<PageResult<User>>('GET', `/api/v1/users${this.buildQuery(opts)}`);
  }

  async assignRole(userId: string, roleId: string): Promise<void> {
    return this.request('POST', `/api/v1/users/${userId}/roles`, { role_id: roleId });
  }

  async removeRole(userId: string, roleId: string): Promise<void> {
    return this.request('DELETE', `/api/v1/users/${userId}/roles/${roleId}`);
  }

  // ---------------------------------------------------------------------------
  // Roles
  // ---------------------------------------------------------------------------

  async createRole(input: CreateRoleInput): Promise<Role> {
    return this.request<Role>('POST', '/api/v1/roles', input);
  }

  async listRoles(opts?: ListOptions): Promise<PageResult<Role>> {
    return this.request<PageResult<Role>>('GET', `/api/v1/roles${this.buildQuery(opts)}`);
  }

  // ---------------------------------------------------------------------------
  // Organizations
  // ---------------------------------------------------------------------------

  async createOrg(input: CreateOrgInput): Promise<Organization> {
    return this.request<Organization>('POST', '/api/v1/organizations', input);
  }

  async listOrgs(opts?: ListOptions): Promise<PageResult<Organization>> {
    return this.request<PageResult<Organization>>('GET', `/api/v1/organizations${this.buildQuery(opts)}`);
  }

  // ---------------------------------------------------------------------------
  // Policy
  // ---------------------------------------------------------------------------

  async checkPermission(userId: string, resource: string, action: string): Promise<PolicyCheckResult> {
    return this.request<PolicyCheckResult>('POST', '/api/v1/policies/check', {
      user_id: userId, resource, action,
    });
  }

  // ---------------------------------------------------------------------------
  // AI Agent Identity (MCP Auth)
  // ---------------------------------------------------------------------------

  /** Register a new AI agent identity. Requires a user access token. */
  async registerAgent(input: RegisterAgentInput, accessToken: string): Promise<Agent> {
    return this.requestWithToken<Agent>('POST', '/api/v1/agents/register', input, accessToken);
  }

  /** List all AI agents for the configured tenant. */
  async listAgents(accessToken: string): Promise<{ agents: Agent[]; total: number }> {
    return this.requestWithToken('GET', '/api/v1/agents', undefined, accessToken);
  }

  /** Exchange a user token for an agent-scoped token (RFC 8693). */
  async exchangeAgentToken(
    agentId: string,
    subjectToken: string,
    scopes?: string[],
    mcpServers?: string[],
  ): Promise<AgentTokenResponse> {
    return this.request<AgentTokenResponse>('POST', '/api/v1/agents/token', {
      subject_token: subjectToken,
      agent_id: agentId,
      scope: scopes,
      mcp_servers: mcpServers,
    });
  }

  /** Verify an agent token and return its claims. */
  async verifyAgentToken(token: string): Promise<AgentTokenClaims> {
    return this.request<AgentTokenClaims>('POST', '/api/v1/agents/verify', { token });
  }

  // ---------------------------------------------------------------------------
  // OAuth/OIDC
  // ---------------------------------------------------------------------------

  /** Fetch the OpenID Connect discovery document. */
  async getOIDCDiscovery(): Promise<Record<string, unknown>> {
    return this.request('GET', '/.well-known/openid-configuration');
  }

  /** Fetch the JWKS for token verification. */
  async getJWKS(): Promise<{ keys: Array<Record<string, unknown>> }> {
    return this.request('GET', '/oauth/jwks');
  }

  /** Fetch UserInfo for an access token. */
  async getUserInfo(accessToken: string): Promise<Record<string, unknown>> {
    return this.requestWithToken('GET', '/oauth/userinfo', undefined, accessToken);
  }

  /** Register an OAuth client via RFC 7591 dynamic registration. */
  async registerOAuthClient(input: {
    client_name: string;
    redirect_uris: string[];
    grant_types?: string[];
    response_types?: string[];
    scope?: string;
  }): Promise<{ client_id: string; client_secret?: string }> {
    return this.request('POST', '/api/v1/oauth/register', input);
  }

  /** List OAuth clients for the current tenant. */
  async listOAuthClients(accessToken: string): Promise<unknown[]> {
    return this.requestWithToken('GET', '/api/v1/oauth/clients', undefined, accessToken);
  }

  /** Delete an OAuth client. */
  async deleteOAuthClient(accessToken: string, clientId: string): Promise<void> {
    return this.requestWithToken('DELETE', `/api/v1/oauth/clients/${clientId}`, undefined, accessToken);
  }

  /** Revoke a token (RFC 7009). */
  async revokeToken(token: string): Promise<void> {
    return this.request('POST', '/api/v1/oauth/revoke', { token });
  }

  /** Initiate device authorization flow (RFC 8628). */
  async deviceAuthorization(clientId: string, scope: string): Promise<{
    device_code: string;
    user_code: string;
    verification_uri: string;
    expires_in: number;
    interval: number;
  }> {
    return this.request('POST', '/api/v1/oauth/device_authorization', { client_id: clientId, scope });
  }

  /** Build an authorization URL with PKCE support. */
  buildAuthorizeURL(opts: {
    client_id: string;
    redirect_uri: string;
    response_type: string;
    scope?: string;
    state?: string;
    nonce?: string;
    code_challenge?: string;
    code_challenge_method?: string;
  }): string {
    const params = new URLSearchParams();
    params.set('client_id', opts.client_id);
    params.set('redirect_uri', opts.redirect_uri);
    params.set('response_type', opts.response_type);
    if (opts.scope) params.set('scope', opts.scope);
    if (opts.state) params.set('state', opts.state);
    if (opts.nonce) params.set('nonce', opts.nonce);
    if (opts.code_challenge) {
      params.set('code_challenge', opts.code_challenge);
      params.set('code_challenge_method', opts.code_challenge_method || 'S256');
    }
    return `${this.config.gatewayUrl}/oauth/authorize?${params.toString()}`;
  }

  // ---------------------------------------------------------------------------
  // Internal
  // ---------------------------------------------------------------------------

  private headers(): Record<string, string> {
    const h: Record<string, string> = {
      'X-Tenant-ID': this.config.tenantId,
      'Content-Type': 'application/json',
    };
    if (this.apiKey) h['X-API-Key'] = this.apiKey;
    return h;
  }

  private buildQuery(opts?: ListOptions): string {
    if (!opts) return '';
    const params = new URLSearchParams();
    if (opts.page) params.set('page', String(opts.page));
    if (opts.page_size) params.set('page_size', String(opts.page_size));
    if (opts.search) params.set('search', opts.search);
    if (opts.status) params.set('status', opts.status);
    const qs = params.toString();
    return qs ? `?${qs}` : '';
  }

  private async requestWithToken<T>(method: string, path: string, body: unknown | undefined, accessToken: string): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const h = this.headers();
      h['Authorization'] = `Bearer ${accessToken}`;
      const resp = await fetch(`${this.config.gatewayUrl}${path}`, {
        method,
        headers: h,
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (!resp.ok) {
        let code = '';
        let message = '';
        try {
          const parsed = await resp.json();
          code = parsed.code || '';
          message = parsed.message || parsed.error || '';
        } catch {
          message = await resp.text().catch(() => `HTTP ${resp.status}`);
        }
        throw new GGIDError(resp.status, message || `HTTP ${resp.status}`, code);
      }

      if (resp.status === 204) return undefined as T;
      return (await resp.json()) as T;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const resp = await fetch(`${this.config.gatewayUrl}${path}`, {
        method,
        headers: this.headers(),
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (!resp.ok) {
        let code = '';
        let message = '';
        try {
          const parsed = await resp.json();
          code = parsed.code || '';
          message = parsed.message || '';
        } catch {
          message = await resp.text().catch(() => `HTTP ${resp.status}`);
        }
        throw new GGIDError(resp.status, message || `HTTP ${resp.status}`, code);
      }

      if (resp.status === 204) return undefined as T;
      return (await resp.json()) as T;
    } finally {
      clearTimeout(timeoutId);
    }
  }
}
