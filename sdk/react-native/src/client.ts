import * as SecureStore from 'expo-secure-store';
import * as AuthSession from 'expo-auth-session';
import * as WebBrowser from 'expo-web-browser';
import type { GGIDConfig, GGIDSession, GGIDUser, GGIDClaims } from './types';

const SESSION_KEY = 'ggid_session';

WebBrowser.maybeCompleteAuthSession();

export class GGIDClient {
  private config: GGIDConfig;
  private discovery: AuthSession.DiscoveryDocument | null = null;

  constructor(config: GGIDConfig) {
    this.config = config;
  }

  /** Get OAuth discovery document */
  async getDiscovery(): Promise<AuthSession.DiscoveryDocument> {
    if (this.discovery) return this.discovery;
    this.discovery = {
      authorizationEndpoint: `${this.config.baseUrl}/api/v1/oauth/authorize`,
      tokenEndpoint: `${this.config.baseUrl}/api/v1/oauth/token`,
      revocationEndpoint: `${this.config.baseUrl}/api/v1/oauth/revoke`,
      userInfoEndpoint: `${this.config.baseUrl}/api/v1/oauth/userinfo`,
    };
    return this.discovery;
  }

  /** Build authorization request */
  private makeAuthRequest(): AuthSession.AuthRequest {
    return new AuthSession.AuthRequest({
      clientId: this.config.clientId,
      redirectUri: this.config.redirectUri,
      scopes: (this.config.scopes || 'openid profile email').split(' '),
      extraParams: {
        tenant_id: this.config.tenantId,
      },
      usePKCE: true,
    });
  }

  /** Start OAuth login flow */
  async login(): Promise<GGIDSession | null> {
    const discovery = await this.getDiscovery();
    const request = this.makeAuthRequest();

    const result = await request.promptAsync(discovery, {
      useProxy: true,
    });

    if (result.type !== 'success') {
      return null;
    }

    const { code } = result.params;
    if (!code) return null;

    // Exchange code for token
    const tokenResult = await AuthSession.exchangeCodeAsync(
      {
        clientId: this.config.clientId,
        code,
        redirectUri: this.config.redirectUri,
        extraParams: {
          tenant_id: this.config.tenantId,
        },
        codeVerifier: request.codeVerifier,
      },
      discovery,
    );

    if (!tokenResult.accessToken) return null;

    // Get user info
    const user = await this.getUserInfo(tokenResult.accessToken);

    const session: GGIDSession = {
      accessToken: tokenResult.accessToken,
      refreshToken: tokenResult.refreshToken,
      expiresAt: Date.now() + (tokenResult.expiresIn || 3600) * 1000,
      user,
    };

    await this.saveSession(session);
    return session;
  }

  /** Get user info from GGID */
  async getUserInfo(accessToken: string): Promise<GGIDUser | null> {
    try {
      const resp = await fetch(
        `${this.config.baseUrl}/api/v1/oauth/userinfo`,
        {
          headers: {
            Authorization: `Bearer ${accessToken}`,
            'X-Tenant-ID': this.config.tenantId,
          },
        }
      );
      if (!resp.ok) return null;
      const data = await resp.json();
      return {
        sub: data.sub,
        name: data.name,
        email: data.email,
        roles: data.roles || [],
        picture: data.picture,
      };
    } catch {
      return null;
    }
  }

  /** Decode JWT claims without verification (for local checks) */
  decodeClaims(token: string): GGIDClaims | null {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) return null;
      const payload = JSON.parse(atob(parts[1]));
      return {
        sub: payload.sub || '',
        tenant_id: payload.tenant_id || '',
        roles: payload.roles || [],
        scope: payload.scope || '',
        exp: payload.exp || 0,
        iat: payload.iat || 0,
        iss: payload.iss || '',
      };
    } catch {
      return null;
    }
  }

  /** Check if current session is valid */
  async getSession(): Promise<GGIDSession | null> {
    const raw = await SecureStore.getItemAsync(SESSION_KEY);
    if (!raw) return null;

    try {
      const session: GGIDSession = JSON.parse(raw);
      if (Date.now() > session.expiresAt) {
        // Try refresh
        if (session.refreshToken) {
          const refreshed = await this.refreshToken(session.refreshToken);
          if (refreshed) return refreshed;
        }
        await this.logout();
        return null;
      }
      return session;
    } catch {
      return null;
    }
  }

  /** Refresh access token */
  async refreshToken(refreshToken: string): Promise<GGIDSession | null> {
    const discovery = await this.getDiscovery();
    try {
      const body = new URLSearchParams({
        grant_type: 'refresh_token',
        refresh_token: refreshToken,
        client_id: this.config.clientId,
        tenant_id: this.config.tenantId,
      });
      const resp = await fetch(discovery.tokenEndpoint!, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: body.toString(),
      });
      if (!resp.ok) return null;
      const data = await resp.json();
      const user = await this.getUserInfo(data.access_token);
      const session: GGIDSession = {
        accessToken: data.access_token,
        refreshToken: data.refresh_token || refreshToken,
        expiresAt: Date.now() + (data.expires_in || 3600) * 1000,
        user,
      };
      await this.saveSession(session);
      return session;
    } catch {
      return null;
    }
  }

  /** Check RBAC permission via GGID policy engine */
  async checkPermission(
    resource: string,
    action: string
  ): Promise<boolean> {
    const session = await this.getSession();
    if (!session) return false;

    const claims = this.decodeClaims(session.accessToken);
    const userId = claims?.sub || '';

    try {
      const resp = await fetch(
        `${this.config.baseUrl}/api/v1/policies/check`,
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${session.accessToken}`,
            'Content-Type': 'application/json',
            'X-Tenant-ID': this.config.tenantId,
          },
          body: JSON.stringify({
            user_id: userId,
            resource,
            action,
          }),
        }
      );
      if (!resp.ok) return false;
      const data = await resp.json();
      return data.allowed === true;
    } catch {
      return false;
    }
  }

  /** Check if user has a specific role */
  async hasRole(role: string): Promise<boolean> {
    const session = await this.getSession();
    if (!session?.user) return false;
    return session.user.roles.includes(role);
  }

  /** Get auth header for API calls */
  async getAuthHeader(): Promise<string | null> {
    const session = await this.getSession();
    if (!session) return null;
    return `Bearer ${session.accessToken}`;
  }

  /** Logout and clear session */
  async logout(): Promise<void> {
    const session = await this.getSession();
    if (session?.accessToken) {
      // Best-effort revoke
      try {
        const discovery = await this.getDiscovery();
        await fetch(discovery.revocationEndpoint!, {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: `token=${session.accessToken}&client_id=${this.config.clientId}`,
        });
      } catch {}
    }
    await SecureStore.deleteItemAsync(SESSION_KEY);
  }

  /** Save session to secure storage */
  private async saveSession(session: GGIDSession): Promise<void> {
    await SecureStore.setItemAsync(SESSION_KEY, JSON.stringify(session));
  }
}
