'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ERPUser, getUser, hasPermission, ERPLayout, authHeader, API_BASE } from '@/lib/auth';
import { PermissionGate } from '@/components/PermissionGate';

export default function OrdersPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);
  const [orders, setOrders] = useState<any[]>([]);

  useEffect(() => {
    const u = getUser(); if (!u) { router.push('/login'); return; } setUser(u);
    fetch(`${API_BASE}/api/orders`, { headers: { ...authHeader() } })
      .then(r => r.json()).then(d => setOrders(d.orders || [])).catch(() => {});
  }, [router]);

  if (!user) return null;
  const canWrite = hasPermission(user, 'orders:write');
  const canApprove = hasPermission(user, 'orders:approve');

  return (
    <PermissionGate user={user} perm="orders:read">
      <ERPLayout user={user}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <h1 style={{ fontSize: 24 }}>Orders</h1>
          <button disabled={!canWrite} style={{ padding: '6px 16px', border: 'none', borderRadius: 4, background: canWrite ? '#1890ff' : '#d9d9d9', color: canWrite ? '#fff' : '#999', cursor: canWrite ? 'pointer' : 'not-allowed' }}>New Order</button>
        </div>
        <table style={{ width: '100%', marginTop: 16, borderCollapse: 'collapse' }}>
          <thead><tr><th style={{ padding: 8, textAlign: 'left' }}>Order ID</th><th style={{ padding: 8 }}>Customer</th><th style={{ padding: 8 }}>Amount</th><th style={{ padding: 8 }}>Status</th>{canApprove && <th style={{ padding: 8 }}>Action</th>}</tr></thead>
          <tbody>{orders.map(o => <tr key={o.id}><td style={{ padding: 8 }}>{o.id}</td><td style={{ padding: 8 }}>{o.customer}</td><td style={{ padding: 8 }}>${o.amount}</td><td style={{ padding: 8, color: o.status === 'approved' ? '#52c41a' : '#faad14' }}>{o.status}</td>{canApprove && <td style={{ padding: 8 }}><button disabled={o.status !== 'pending'} style={{ padding: '4px 12px', fontSize: 12, background: o.status === 'pending' ? '#52c41a' : '#d9d9d9', color: '#fff', border: 'none', borderRadius: 3 }}>Approve</button></td>}</tr>)}</tbody>
        </table>
      </ERPLayout>
    </PermissionGate>);
}
