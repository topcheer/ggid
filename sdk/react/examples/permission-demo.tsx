/**
 * GGID React SDK Demo — OAuth + SAML with Fine-Grained Permissions
 *
 * Usage in your React app:
 * ```tsx
 * import { PermissionApp } from '@ggid/react/demo';
 * ```
 */
import React, { useState, useEffect, ReactNode } from 'react';
import { usePasskey } from '../src/passkey';

// === Types ===
interface DemoUser {
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
}

// === Permission helpers ===
export function hasPermission(user: DemoUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes('admin')) return true;
  return user.permissions.includes(perm);
}

// === Components ===
export function PermissionGate({ user, perm, children, fallback }: {
  user: DemoUser | null;
  perm: string;
  children: ReactNode;
  fallback?: ReactNode;
}) {
  if (!hasPermission(user, perm)) {
    return <>{fallback || <Forbidden403 perm={perm} />}</>;
  }
  return <>{children}</>;
}

export function Forbidden403({ perm }: { perm: string }) {
  return (
    <div style={{ textAlign: 'center', padding: '40px' }}>
      <h1>403 Forbidden</h1>
      <p>You need permission: <code>{perm}</code></p>
    </div>
  );
}

export function RoleBadge({ role }: { role: string }) {
  const colors: Record<string, string> = {
    admin: '#f5222d', sales_manager: '#1890ff',
    warehouse_manager: '#52c41a', finance_officer: '#faad14',
  };
  return (
    <span style={{
      background: colors[role] || '#d9d9d9', color: '#fff',
      padding: '2px 8px', borderRadius: '4px', fontSize: '12px', marginRight: '4px',
    }}>{role.replace(/_/g, ' ').toUpperCase()}</span>
  );
}

export function SideMenu({ user }: { user: DemoUser }) {
  const items: { href: string; label: string; perm?: string }[] = [
    { href: '/dashboard', label: 'Dashboard' },
    { href: '/orders', label: 'Orders', perm: 'orders:read' },
    { href: '/inventory', label: 'Inventory', perm: 'inventory:read' },
    { href: '/admin', label: 'Admin', perm: 'admin' },
  ];

  return (
    <aside style={{ width: 220, padding: 16, borderRight: '1px solid #e8e8e8' }}>
      <h3>GGID Demo</h3>
      <div style={{ marginBottom: 16 }}>
        {user.roles.map(r => <RoleBadge key={r} role={r} />)}
      </div>
      <ul style={{ listStyle: 'none', padding: 0 }}>
        {items
          .filter(item => !item.perm || hasPermission(user, item.perm))
          .map(item => (
            <li key={item.href} style={{ marginBottom: 8 }}>
              <a href={item.href}>{item.label}</a>
            </li>
          ))}
      </ul>
    </aside>
  );
}

export function Dashboard({ user }: { user: DemoUser }) {
  return (
    <div style={{ display: 'flex' }}>
      <SideMenu user={user} />
      <main style={{ flex: 1, padding: 24 }}>
        <h1>Dashboard</h1>
        <p>Welcome, {user.username}</p>
        <h3>Your Permissions:</h3>
        <ul>
          {user.permissions.map(p => <li key={p}><code>{p}</code></li>)}
        </ul>
      </main>
    </div>
  );
}

export function InventoryPage({ user }: { user: DemoUser }) {
  const canWrite = hasPermission(user, 'inventory:write');
  const canDelete = hasPermission(user, 'inventory:delete');
  return (
    <PermissionGate user={user} perm="inventory:read">
      <div style={{ display: 'flex' }}>
        <SideMenu user={user} />
        <main style={{ flex: 1, padding: 24 }}>
          <h1>Inventory</h1>
          {canWrite && <button style={{ marginRight: 8 }}>New Item</button>}
          {canDelete && <button style={{ color: 'red' }}>Delete</button>}
          {!canWrite && !canDelete && <p>Read-only access</p>}
          <table style={{ width: '100%', marginTop: 16 }}>
            <thead><tr><th>SKU</th><th>Name</th><th>Qty</th></tr></thead>
            <tbody><tr><td colSpan={3}>No data</td></tr></tbody>
          </table>
        </main>
      </div>
    </PermissionGate>
  );
}

export function OrdersPage({ user }: { user: DemoUser }) {
  const canWrite = hasPermission(user, 'orders:write');
  const canApprove = hasPermission(user, 'orders:approve');
  return (
    <PermissionGate user={user} perm="orders:read">
      <div style={{ display: 'flex' }}>
        <SideMenu user={user} />
        <main style={{ flex: 1, padding: 24 }}>
          <h1>Orders</h1>
          {canWrite && <button style={{ marginRight: 8 }}>New Order</button>}
          {canApprove && <button>Approve</button>}
          {!canWrite && !canApprove && <p>Read-only access</p>}
        </main>
      </div>
    </PermissionGate>
  );
}

export function AdminPage({ user }: { user: DemoUser }) {
  return (
    <PermissionGate user={user} perm="admin">
      <div style={{ display: 'flex' }}>
        <SideMenu user={user} />
        <main style={{ flex: 1, padding: 24 }}>
          <h1>Admin Panel</h1>
          <p>Admin-only content</p>
        </main>
      </div>
    </PermissionGate>
  );
}
