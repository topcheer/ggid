/**
 * GGID OAuth Client for React Native (Expo)
 * 
 * Implements the authorization code flow using expo-web-browser.
 * Flow: open browser → user logs in → redirect back with code → exchange token → get userinfo
 */

import * as WebBrowser from 'expo-web-browser';
import * as SecureStore from 'expo-secure-store';

// ─── Configuration ───────────────────────────────────────────
const GGID_URL = process.env.EXPO_PUBLIC_GGID_URL || 'https://ggid.iot2.win';
const CLIENT_ID = process.env.EXPO_PUBLIC_CLIENT_ID || 'gcid__sbYZX3_2aJ4eDz-Oy1qRQ';
const REDIRECT_URI = process.env.EXPO_PUBLIC_REDIRECT_URI || 'exp://localhost:8081/+redirect';
const TENANT_ID = '00000000-0000-0000-0000-000000000001';
const SCOPES = 'openid profile email';

// PKCE helpers
function generateRandomString(length: number): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

async function sha256(plain: string): Promise<string> {
  // Use SubtleCrypto if available (Expo web), otherwise simple hash fallback
  try {
    const crypto = globalThis.crypto;
    if (crypto?.subtle) {
      const encoder = new TextEncoder();
      const data = encoder.encode(plain);
      const hash = await crypto.subtle.digest('SHA-256', data);
      return btoa(String.fromCharCode(...new Uint8Array(hash)))
        .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    }
  } catch {}
  // Fallback: simple base64 (not cryptographically secure, but works for demo)
  return btoa(plain).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// ─── Types ──────────────────────────────────────────────────
export interface UserInfo {
  sub: string;
  email?: string;
  email_verified?: boolean;
  name?: string;
  preferred_username?: string;
  picture?: string;
  tenant_id?: string;
}

export interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
  id_token?: string;
  scope?: string;
}

export interface JWTClaims {
  sub?: string;
  iss?: string;
  aud?: string;
  exp?: number;
  iat?: number;
  email?: string;
  scope?: string;          // OAuth scopes only (openid, profile, email)
  permissions?: string[];  // Fine-grained permissions (inventory:read)
  tenant_id?: string;
  roles?: string[];
  [key: string]: any;
}

// ─── Storage ────────────────────────────────────────────────
const TOKEN_KEY = 'ggid_access_token';
const REFRESH_KEY = 'ggid_refresh_token';
const USERINFO_KEY = 'ggid_userinfo';

export async function saveSession(token: TokenResponse, userInfo: UserInfo): Promise<void> {
  await SecureStore.setItemAsync(TOKEN_KEY, token.access_token);
  if (token.refresh_token) {
    await SecureStore.setItemAsync(REFRESH_KEY, token.refresh_token);
  }
  await SecureStore.setItemAsync(USERINFO_KEY, JSON.stringify(userInfo));
}

export async function getStoredToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(TOKEN_KEY);
  } catch {
    return null;
  }
}

export async function getStoredUserInfo(): Promise<UserInfo | null> {
  try {
    const raw = await SecureStore.getItemAsync(USERINFO_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export async function clearSession(): Promise<void> {
  await SecureStore.deleteItemAsync(TOKEN_KEY);
  await SecureStore.deleteItemAsync(REFRESH_KEY);
  await SecureStore.deleteItemAsync(USERINFO_KEY);
}

// ─── OAuth Flow ─────────────────────────────────────────────

/**
 * Step 1: Open browser for user to log in at GGID
 * Returns the authorization code from the redirect URL
 */
export async function authorize(): Promise<string | null> {
  const state = generateRandomString(32);
  const codeVerifier = generateRandomString(64);
  const codeChallenge = await sha256(codeVerifier);

  // Build authorize URL
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    response_type: 'code',
    scope: SCOPES,
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  });

  const authorizeUrl = `${GGID_URL}/oauth/authorize?${params.toString()}`;

  // Open system browser for auth
  const result = await WebBrowser.openAuthSessionAsync(authorizeUrl, REDIRECT_URI);

  if (result.type !== 'success') {
    return null;
  }

  // Parse redirect URL for code and state
  const redirectUrl = new URL(result.url);
  const code = redirectUrl.searchParams.get('code');
  const returnedState = redirectUrl.searchParams.get('state');

  if (!code || returnedState !== state) {
    throw new Error('Invalid OAuth redirect: code or state mismatch');
  }

  // Store code_verifier for token exchange
  await SecureStore.setItemAsync('ggid_code_verifier', codeVerifier);

  return code;
}

/**
 * Step 2: Exchange authorization code for access token
 */
export async function exchangeToken(code: string): Promise<TokenResponse> {
  const codeVerifier = await SecureStore.getItemAsync('ggid_code_verifier');
  await SecureStore.deleteItemAsync('ggid_code_verifier');

  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    ...(codeVerifier ? { code_verifier: codeVerifier } : {}),
  });

  const res = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'X-Tenant-ID': TENANT_ID,
    },
    body: body.toString(),
  });

  if (!res.ok) {
    const err = await res.text();
    throw new Error(`Token exchange failed: ${res.status} ${err}`);
  }

  return res.json();
}

/**
 * Step 3: Get user info using access token
 */
export async function getUserInfo(accessToken: string): Promise<UserInfo> {
  const res = await fetch(`${GGID_URL}/api/v1/oauth/userinfo`, {
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'X-Tenant-ID': TENANT_ID,
    },
  });

  if (!res.ok) {
    throw new Error(`UserInfo failed: ${res.status}`);
  }

  return res.json();
}

/**
 * Parse JWT token to extract claims (without verification)
 */
export function parseJWT(token: string): JWTCclaims {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return {};
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload);
  } catch {
    return {};
  }
}

/**
 * Full login flow: authorize → exchange token → get userinfo
 */
export async function login(): Promise<{ token: TokenResponse; userInfo: UserInfo }> {
  const code = await authorize();
  if (!code) throw new Error('Authorization cancelled');

  const token = await exchangeToken(code);
  const userInfo = await getUserInfo(token.access_token);
  await saveSession(token, userInfo);

  return { token, userInfo };
}

/**
 * Logout: clear local session
 */
export async function logout(): Promise<void> {
  await clearSession();
}

export const GGID_CONFIG = {
  url: GGID_URL,
  clientId: CLIENT_ID,
  redirectUri: REDIRECT_URI,
  tenantId: TENANT_ID,
  scopes: SCOPES,
};
