/**
 * GGID OAuth + Permission helpers
 */

export const GGID_URL = process.env.NEXT_PUBLIC_GGID_URL || 'https://ggid.iot2.win';
export const CLIENT_ID = process.env.NEXT_PUBLIC_CLIENT_ID || '';
export const CLIENT_SECRET = process.env.CLIENT_SECRET || '';
export const REDIRECT_URI = process.env.NEXT_PUBLIC_REDIRECT_URI || 'https://erp.iot2.win/api/auth/callback';

export interface ERPUser {
  username: string;
  email: string;
  displayName: string;
  roles: string[];
  permissions: string[];
}

/** Check if user has a specific permission */
export function hasPermission(user: ERPUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes('admin')) return true;
  return user.permissions.includes(perm);
}

/** Parse JWT to extract user info + permissions */
export function parseJWT(token: string): ERPUser | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    const scopes = payload.scope?.split(' ') || payload.scopes || [];
    return {
      username: payload.username || payload.preferred_username || 'user',
      email: payload.email || '',
      displayName: payload.name || payload.username || 'User',
      roles: payload.roles || scopes.filter((s: string) => !s.includes(':')),
      permissions: scopes,
    };
  } catch {
    return null;
  }
}

/** Get user from localStorage token */
export function getUser(): ERPUser | null {
  if (typeof window === 'undefined') return null;
  const token = localStorage.getItem('erp_access_token');
  if (!token) return null;
  return parseJWT(token);
}

/** Get auth header */
export function authHeader(): Record<string, string> {
  if (typeof window === 'undefined') return {};
  const token = localStorage.getItem('erp_access_token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/** Logout */
export function logout() {
  localStorage.removeItem('erp_access_token');
  window.location.href = '/login';
}
