/**
 * Cross-Board ERP Demo — React/Next.js Frontend
 * Uses GGID OAuth2 Authorization Code + PKCE flow.
 * Tenant: 00000000-0000-0000-0000-000000000003
 */

export const GGID_URL = process.env.NEXT_PUBLIC_GGID_URL || 'https://ggid.iot2.win';
export const CLIENT_ID = process.env.NEXT_PUBLIC_CLIENT_ID || '';
export const REDIRECT_URI = process.env.NEXT_PUBLIC_REDIRECT_URI || 'http://localhost:3300/callback';
export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:3200';
const TENANT = '00000003-0000-0000-0000-000000000001';

export interface ERPUser {
  user_id: string;
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
}

export function hasPermission(user: ERPUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes('admin')) return true;
  return user.permissions.includes(perm) || user.permissions.includes(`${perm}:all`);
}

export function parseJWT(token: string): ERPUser | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    const perms = payload.permissions || [];
    return {
      user_id: payload.sub || payload.user_id || '',
      username: payload.username || payload.preferred_username || 'user',
      email: payload.email || '',
      roles: payload.roles || [],
      permissions: perms,
    };
  } catch { return null; }
}

export function getUser(): ERPUser | null {
  if (typeof window === 'undefined') return null;
  const token = localStorage.getItem('erp_token');
  return token ? parseJWT(token) : null;
}

export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('erp_token');
}

export function authHeader(): Record<string, string> {
  const t = getToken();
  const h: Record<string, string> = { 'X-Tenant-ID': TENANT };
  if (t) h['Authorization'] = `Bearer ${t}`;
  return h;
}

export function logout() {
  localStorage.removeItem('erp_token');
  localStorage.removeItem('erp_pkce_verifier');
  window.location.href = '/login';
}

/** Generate PKCE code_verifier and code_challenge (S256) */
export function generatePKCE(): { verifier: string; challenge: string } {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  const verifier = btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');

  // For S256 challenge, use SubtleCrypto (async in browser)
  // We store verifier and compute challenge in buildAuthUrl
  return { verifier, challenge: '' };
}

/** Build OAuth2 Authorization Code + PKCE URL */
export async function buildAuthUrl(): Promise<string> {
  const { verifier } = generatePKCE();
  localStorage.setItem('erp_pkce_verifier', verifier);

  // Compute S256 challenge
  const encoder = new TextEncoder();
  const digest = await crypto.subtle.digest('SHA-256', encoder.encode(verifier));
  const challenge = btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');

  const params = new URLSearchParams({
    response_type: 'code',
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'openid profile email',
    state: 'erp-react-pkce',
    code_challenge: challenge,
    code_challenge_method: 'S256',
  });

  return `${GGID_URL}/api/v1/oauth/authorize?${params.toString()}`;
}

/** Exchange authorization code for token (with PKCE verifier) */
export async function exchangeCodeForToken(code: string): Promise<string | null> {
  const verifier = localStorage.getItem('erp_pkce_verifier');
  if (!verifier) return null;

  const res = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': TENANT },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      code_verifier: verifier,
    }),
  });

  if (!res.ok) return null;
  const data = await res.json();
  localStorage.setItem('erp_token', data.access_token);
  localStorage.removeItem('erp_pkce_verifier');
  return data.access_token;
}