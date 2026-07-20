'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ERPUser, getUser, hasPermission } from '../../lib/auth';
import { ERPLayout } from '../../components/ERPLayout';

export default function DashboardPage() {
  const router = useRouter();
  const [user, setUser] = useState(null);
  useEffect(() => { const u = getUser(); if (!u) { router.push('/login'); return; } setUser(u); }, [router]);
  if (!user) return null;

  const stats = [
    { label: 'Total Orders', value: 1247, perm: 'orders:read' },
    { label: 'Inventory Items', value: 458, perm: 'inventory:read' },
    { label: 'Active Users', value: 23, perm: 'users:read' },
  ].filter(s => hasPermission(user, s.perm));

  return (
    <ERPLayout user={user}>
      <h1 style={{ fontSize: 24, fontWeight: 'bold' }}>Dashboard</h1>
      <p style={{ color: '#666' }}>Welcome, {user.username}</p>
      <div style={{ display: 'flex', gap: 16, marginTop: 24 }}>
        {stats.map(s => (
          <div key={s.label} style={{ flex: 1, background: '#f8f9fa', borderRadius: 8, padding: 16, textAlign: 'center' }}>
            <h3 style={{ fontSize: 24, color: '#1890ff', margin: '0 0 4px' }}>{s.value}</h3>
            <p style={{ fontSize: 12, color: '#999', margin: 0 }}>{s.label}</p>
          </div>
        ))}
      </div>
      <h3 style={{ marginTop: 32 }}>Your Permissions</h3>
      <ul style={{ fontSize: 14 }}>
        {user.permissions.map(p => <li key={p}><code>{p}</code></li>)}
      </ul>
    </ERPLayout>
  );
}