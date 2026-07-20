'use client';
import React from 'react';
import { ERPUser, hasPermission, logout } from '@/lib/auth';

export function ERPLayout({ user, children }: { user: ERPUser; children: React.ReactNode }) {
  const menuItems = [
    { href: '/dashboard', label: 'Dashboard', perm: null },
    { href: '/inventory', label: 'Inventory', perm: 'inventory:read' },
    { href: '/orders', label: 'Orders', perm: 'orders:read' },
    { href: '/users', label: 'Users', perm: 'users:read' },
    { href: '/roles', label: 'Roles', perm: 'roles:read' },
    { href: '/orgs', label: 'Organizations', perm: 'orgs:read' },
    { href: '/audit', label: 'Audit Log', perm: 'audit:read' },
  ].filter(item => !item.perm || hasPermission(user, item.perm));

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <aside style={{ width: 240, background: '#1a1a2e', color: '#fff', padding: 20 }}>
        <h2 style={{ margin: '0 0 12px' }}>Cross-Board ERP</h2>
        <div style={{ marginBottom: 16 }}>
          {user.roles.map(r => (
            <span key={r} style={{
              display: 'inline-block', padding: '2px 8px', borderRadius: 4,
              fontSize: 11, marginRight: 4, background: r === 'Admin' ? '#f5222d' : r === 'Manager' ? '#faad14' : '#1890ff',
            }}>{r}</span>
          ))}
        </div>
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {menuItems.map(item => (
            <li key={item.href} style={{ marginBottom: 8 }}>
              <a href={item.href} style={{ color: '#fff', textDecoration: 'none', fontSize: 14 }}>{item.label}</a>
            </li>
          ))}
        </ul>
        <hr style={{ borderColor: '#333', margin: '20px 0' }} />
        <div style={{ fontSize: 12, color: '#999' }}>
          <p>{user.username}</p>
          <button onClick={logout} style={{ color: '#ff4d4f', background: 'none', border: 'none', cursor: 'pointer', fontSize: 12 }}>Logout</button>
        </div>
      </aside>
      <main style={{ flex: 1, padding: 24 }}>{children}</main>
    </div>;
}
