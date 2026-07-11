/**
 * TokenManager — automatic token refresh for Node.js applications.
 *
 * Stores a TokenSet after login and transparently refreshes the access token
 * when it is about to expire (30s before expiry), using the stored refresh token.
 *
 * Usage:
 *
 * ```ts
 * import { GGIDClient, TokenManager } from '@ggid/sdk';
 *
 * const client = new GGIDClient({ gatewayUrl: 'https://iam.example.com' });
 * const tm = new TokenManager(client);
 *
 * await tm.login({ username: 'admin', password: 'Admin@123456' });
 *
 * // Later — auto-refreshes if needed
 * const token = await tm.getAccessToken();
 * await client.listUsers(); // uses the token internally
 * ```
 */

import type { TokenSet, LoginInput } from './types';
import { GGIDClient } from './client';

const REFRESH_MARGIN_MS = 30_000; // 30 seconds before expiry

export class TokenManager {
  private client: GGIDClient;
  private tokens: TokenSet | null = null;
  private obtainedAt = 0; // epoch ms when tokens were obtained
  private refreshPromise: Promise<string> | null = null; // dedup concurrent refreshes

  constructor(client: GGIDClient) {
    this.client = client;
  }

  /** Authenticate and store tokens for auto-refresh. */
  async login(input: LoginInput): Promise<TokenSet> {
    this.tokens = await this.client.login(input);
    this.obtainedAt = Date.now();
    return this.tokens;
  }

  /** Manually set tokens (e.g. restored from a previous session). */
  setTokens(tokens: TokenSet): void {
    this.tokens = tokens;
    this.obtainedAt = Date.now();
  }

  /**
   * Returns a valid access token. If the current token is about to expire
   * (within REFRESH_MARGIN_MS), it transparently refreshes using the stored
   * refresh token.
   *
   * Concurrent calls share a single refresh request.
   */
  async getAccessToken(): Promise<string> {
    if (!this.tokens) {
      throw new Error('ggid: no tokens stored — call login() first');
    }

    // Check if token is still valid with margin
    const expiryMs = this.obtainedAt + (this.tokens.expires_in * 1000);
    if (Date.now() + REFRESH_MARGIN_MS < expiryMs) {
      return this.tokens.access_token;
    }

    // Token is about to expire — refresh
    if (!this.tokens.refresh_token) {
      throw new Error('ggid: no refresh token available for auto-refresh');
    }

    // Dedup: if a refresh is already in progress, wait for it
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefresh();
    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  /** Returns the current token set (may be expired). */
  getTokens(): TokenSet | null {
    return this.tokens;
  }

  /** Clears stored tokens. */
  clear(): void {
    this.tokens = null;
    this.obtainedAt = 0;
    this.refreshPromise = null;
  }

  /** Performs the actual refresh and updates stored tokens. */
  private async doRefresh(): Promise<string> {
    if (!this.tokens?.refresh_token) {
      throw new Error('ggid: no refresh token available');
    }

    const newTokens = await this.client.refreshToken(this.tokens.refresh_token);
    this.tokens = newTokens;
    this.obtainedAt = Date.now();
    return newTokens.access_token;
  }
}
