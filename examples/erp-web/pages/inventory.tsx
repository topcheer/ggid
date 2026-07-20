import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import { UserSession, hasPermission } from '../lib/auth';
import AppLayout from '../components/Layout';
import { Table, Button, Result, Space } from 'antd';

export default function Inventory() {
  const router = useRouter();
  const [session, setSession] = useState<UserSession | null>(null);
  useEffect(() => {
    const raw = localStorage.getItem('ggid_session');
    if (!raw) { router.push('/'); return; }
    setSession(JSON.parse(raw));
  }, [router]);
  if (!session) return <div>Loading...</div>;
  if (!hasPermission(session, 'inventory:read')) {
    return <AppLayout session={session} activeKey="inventory"><Result status="403" title="403" subTitle="No permission for inventory." /></AppLayout>;
  }
  const canWrite = hasPermission(session, 'inventory:write');
  const columns = [
    { title: 'SKU', dataIndex: 'sku', key: 'sku' },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Stock', dataIndex: 'stock', key: 'stock' },
    ...(canWrite ? [{ title: 'Actions', key: 'actions', render: () => (<Space><Button size="small">Edit</Button><Button size="small" danger>Delete</Button></Space>) }] : []),
  ];
  const data = [
    { key: '1', sku: 'SKU-001', name: 'Widget A', stock: 150 },
    { key: '2', sku: 'SKU-002', name: 'Widget B', stock: 75 },
    { key: '3', sku: 'SKU-003', name: 'Gadget C', stock: 12 },
  ];
  return (
    <AppLayout session={session} activeKey="inventory">
      <h1>Inventory</h1>
      {canWrite && <Button type="primary" style={{ marginBottom: 16 }}>+ New Item</Button>}
      {!canWrite && <p style={{ color: '#888' }}>Read-only access.</p>}
      <Table columns={columns} dataSource={data} />
    </AppLayout>
  );
}
