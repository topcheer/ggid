'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ERPUser, getUser, hasPermission, authHeader, API_BASE } from '../../lib/auth';
import { ERPLayout } from '../../components/ERPLayout';
import { PermissionGate } from '../../components/PermissionGate';

export default function InventoryPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);
  const [items, setItems] = useState<any[]>([]);

  useEffect(() => {
    const u = getUser(); if (!u) { router.push('/login'); return; } setUser(u);
    fetch(`${API_BASE}/api/inventory`, { headers: { ...authHeader() } })
      .then(r => r.json()).then(d => setItems(d.items || [])).catch(() => {});
  }, [router]);

  if (!user) return null;
  const canWrite = hasPermission(user, 'inventory:write');
  const canDelete = hasPermission(user, 'inventory:delete');

  return (
    <PermissionGate user={user} perm="inventory:read">
      <ERPLayout user={user}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <h1 style={{ fontSize: 24 }}>Inventory</h1>
          <button disabled={!canWrite} style={{ padding: '6px 16px', border: 'none', borderRadius: 4, background: canWrite ? '#1890ff' : '#d9d9d9', color: canWrite ? '#fff' : '#999', cursor: canWrite ? 'pointer' : 'not-allowed' }}>New Item</button>
        </div>
        <table style={{ width: '100%', marginTop: 16, borderCollapse: 'collapse' }}>
          <thead><tr><th style={{ padding: 8, textAlign: 'left', borderBottom: '1px solid #eee' }}>SKU</th><th style={{ padding: 8 }}>Name</th><th style={{ padding: 8 }}>Qty</th>{canDelete && <th style={{ padding: 8 }}>Action</th>}</tr></thead>
          <tbody>{items.map(i => <tr key={i.id}><td style={{ padding: 8 }}>{i.sku}</td><td style={{ padding: 8 }}>{i.name}</td><td style={{ padding: 8 }}>{i.qty}</td>{canDelete && <td style={{ padding: 8 }}><button disabled={!canDelete} style={{ color: canDelete ? '#ff4d4f' : '#ccc' }}>Delete</button></td>}</tr>)}</tbody>
        </table>
      </ERPLayout>
    </PermissionGate>
  );
}