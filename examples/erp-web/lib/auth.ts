// GGID auth helpers — OAuth flow + JWT scope extraction + permission checks

export const GGID_URL = process.env.GGID_URL || 'https://ggid.iot2.win';
export const CLIENT_ID = process.env.CLIENT_ID || '';
export const CLIENT_SECRET = process.env.CLIENT_SECRET || '';
export const TENANT_ID = process.env.TENANT_ID || '00000000-0000-0000-0000-000000000001';
export const REDIRECT_URI = process.env.REDIRECT_URI || 'http://localhost:3001/callback';

// Permission matrix per role
const ROLE_PERMISSIONS: Record<string, string[]> = {
  'Sales Manager': ['orders:read', 'orders:write', 'orders:approve', 'inventory:read', 'reports:read', 'dashboard:read'],
  'Warehouse Manager': ['orders:read', 'orders:write', 'inventory:read', 'inventory:write', 'inventory:delete', 'reports:read', 'dashboard:read'],
  'Finance Officer': ['orders:read', 'reports:read', 'reports:write', 'audit:read', 'dashboard:read'],
  'Administrator': ['*'],
};

// Map JWT scope display names to role names
const SCOPE_TO_ROLE: Record<string, string> = {
  'platform administrator': 'Administrator',
  'tenant administrator': 'Administrator',
  'administrator': 'Administrator',
  'sales manager': 'Sales Manager',
  'warehouse manager': 'Warehouse Manager',
  'finance officer': 'Finance Officer',
};

export function extractRole(scopes: string[]): string {
  for (const scope of scopes) {
    const lower = scope.toLowerCase();
    if (SCOPE_TO_ROLE[lower]) return SCOPE_TO_ROLE[lower];
  }
  return 'Viewer';
}

export function getPermissions(role: string): string[] {
  if (role === 'Administrator') return ['*'];
  return ROLE_PERMISSIONS[role] || [];
}

export function hasPermission(role: string, resource: string, action: string): boolean {
  const perms = getPermissions(role);
  if (perms.includes('*')) return true;
  return perms.includes(`${resource}:${action}`);
}

// Decode JWT payload (no signature verification — backend enforces)
export function decodeJWTPayload(token: string): Record<string, any> {
  try {
    const parts = token.split('.');
    if (parts.length < 2) return {};
    const payload = Buffer.from(parts[1], 'base64url').toString('utf-8');
    return JSON.parse(payload);
  } catch {
    return {};
  }
}

export function getRoleFromToken(token: string): string {
  const claims = decodeJWTPayload(token);
  const scopes: string[] = claims.scopes || [];
  return extractRole(scopes);
}
