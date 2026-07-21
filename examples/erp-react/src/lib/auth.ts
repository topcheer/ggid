/**
 * Cross-Board ERP Demo — React/Next.js Frontend
 * Uses GGID OAuth2 Authorization Code + PKCE flow.
 * Tenant: 00000000-0000-0000-0000-000000000003
 *
 * SECURITY: Token verification via backend introspect (not inline decode).
 */

export const GGID_URL = process.env.NEXT_PUBLIC_GGID_URL || 'https://ggid.iot2.win';
export const CLIENT_ID = process.env.NEXT_PUBLIC_CLIENT_ID || '';
export const REDIRECT_URI = process.env.NEXT_PUBLIC_REDIRECT_URI || 'http://localhost:3300/callback';
export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:3200';
const TENANT = '00000000-0000-0000-0000-000000000003';

export interface ERPUser {
  user_id: string;
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
}

// Cached verified user
let cachedUser: ERPUser | null = null;
let cachedToken: string | null = null;

export function hasPermission(user: ERPUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes('admin')) return true;
  return user.permissions.includes(perm) || user.permissions.includes(`${perm}:all`);
}

/**
 * Verify token via backend introspect endpoint (not inline decode).
 * This ensures the token is valid and not tampered with.
 */
export async function verifyToken(token: string): Promise<ERPUser | null> {
  if (cachedToken === token && cachedUser) return cachedUser;

  try {
    // Call backend introspect to verify token server-side
    const res = await fetch(`${GGID_URL}/api/v1/oauth/introspect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': TENANT },
      body: new URLSearchParams({ token }),
    });

    if (!res.ok) return null;
    const data = await res.json();

    if (!data.active) return null;

    const user: ERPUser = {
      user_id: data.sub || data.user_id || '',
      username: data.username || data.preferred_username || 'user',
      email: data.email || '',
      roles: data.roles || [],
      permissions: data.permissions || data.scope?.split(' ') || [],
    };

    cachedToken = token;
    cachedUser = user;
    return user;
  } catch {
    return null;
  }
}

/**
 * Get current user — tries cache first, falls back to introspect.
 * For synchronous access (initial render), returns cached user.
 */
export function getUser(): ERPUser | null {
  if (typeof window === 'undefined') return null;
  // Return cached user for synchronous access
  if (cachedUser) return cachedUser;
  // If no cache but token exists, return null (async verify needed)
  const token = getToken();
  if (!token) return null;
  // Trigger async verification
  verifyToken(token).then(u => { if (u) cachedUser = u; });
  return null;
}

/** Async version of getUser that always verifies */
export async function getUserAsync(): Promise<ERPUser | null> {
  if (typeof window === 'undefined') return null;
  const token = getToken();
  if (!token) return null;
  return verifyToken(token);
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
  cachedUser = null;
  cachedToken = null;
  window.location.href = '/login';
}

/** Generate PKCE code_verifier */
export function generatePKCE(): { verifier: string; challenge: string } {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  const verifier = btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  return { verifier, challenge: '' };
}

/** Build OAuth2 Authorization Code + PKCE URL */
export async function buildAuthUrl(): Promise<string> {
  const { verifier } = generatePKCE();
  localStorage.setItem('erp_pkce_verifier', verifier);

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

  // Verify the token immediately
  const user = await verifyToken(data.access_token);
  if (user) cachedUser = user;

  return data.access_token;
}