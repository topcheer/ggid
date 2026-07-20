/**
 * Cross-Board ERP Demo — React/Next.js Frontend
 * Uses GGID React SDK for OAuth + fine-grained permissions.
 */

export const GGID_URL = process.env.NEXT_PUBLIC_GGID_URL || 'https://ggid.iot2.win';
export const CLIENT_ID = process.env.NEXT_PUBLIC_CLIENT_ID || '';
export const CLIENT_SECRET = process.env.CLIENT_SECRET || '';
export const REDIRECT_URI = process.env.NEXT_PUBLIC_REDIRECT_URI || 'http://localhost:3300/callback';
export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:3200';

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
    const perms = payload.permissions || payload.scope?.split(' ') || [];
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
  return t ? { Authorization: `Bearer ${t}` } : {};
}

export function logout() {
  localStorage.removeItem('erp_token');
  window.location.href = '/login';
}
