/** GGID ERP Web — OAuth config + permission helpers */

export const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
export const ERP_API = process.env.ERP_API || 'http://localhost:8090';
export const CLIENT_ID = process.env.CLIENT_ID || '';
export const CLIENT_SECRET = process.env.CLIENT_SECRET || '';
export const REDIRECT_URI = process.env.REDIRECT_URI || 'http://localhost:3000/auth/callback';

export interface UserSession {
  access_token: string;
  username: string;
  email: string;
  display_name: string;
  scopes: string[];
  roles: string[];
}

/** Decode JWT payload without verification (client-side only). */
export function decodeJWT(token: string): Record<string, any> | null {
  try {
    const parts = token.split('.');
    if (parts.length < 2) return null;
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const decoded = Buffer.from(payload, 'base64').toString('utf-8');
    return JSON.parse(decoded);
  } catch {
    return null;
  }
}

/** Check if user has a specific permission based on roles. */
export function hasPermission(session: UserSession | null, perm: string): boolean {
  if (!session) return false;
  if (session.scopes.includes('platform:admin') || session.scopes.includes('admin') || session.scopes.includes('tenant:admin')) {
    return true;
  }
  const roles = session.roles.map(r => r.toLowerCase());
  switch (perm) {
    case 'inventory:read':
      return roles.some(r => ['warehouse_manager', 'sales_manager', 'erp_admin', 'erp:system_admin'].includes(r));
    case 'inventory:write':
      return roles.some(r => ['warehouse_manager', 'erp_admin', 'erp:system_admin'].includes(r));
    case 'orders:read':
      return true;
    case 'orders:write':
      return roles.some(r => ['sales_manager', 'warehouse_manager', 'erp_admin', 'erp:system_admin'].includes(r));
    case 'orders:approve':
      return roles.some(r => ['sales_manager', 'erp_admin', 'erp:system_admin'].includes(r));
    case 'reports:read':
      return roles.some(r => ['sales_manager', 'finance_officer', 'erp_admin', 'erp:system_admin'].includes(r));
    case 'admin':
      return session.scopes.includes('platform:admin') || session.scopes.includes('admin');
    default:
      return false;
  }
}

/** Build sidebar menu items based on permissions. */
export function getMenuItems(session: UserSession | null) {
  const items: { key: string; label: string; href?: string }[] = [
    { key: 'dashboard', label: 'Dashboard', href: '/dashboard' },
  ];
  if (hasPermission(session, 'orders:read')) {
    items.push({ key: 'orders', label: 'Orders', href: '/orders' });
  }
  if (hasPermission(session, 'inventory:read')) {
    items.push({ key: 'inventory', label: 'Inventory', href: '/inventory' });
  }
  if (hasPermission(session, 'reports:read')) {
    items.push({ key: 'reports', label: 'Reports', href: '/reports' });
  }
  if (hasPermission(session, 'admin')) {
    items.push({ key: 'admin', label: 'Admin', href: '/admin' });
  }
  return items;
}
