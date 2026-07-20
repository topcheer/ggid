import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { Card, Table, Button, Tag } from 'antd';
import { getRoleFromToken, hasPermission } from '../lib/auth';

const mockOrders = [
  { key: '1', id: 'ORD-001', customer: 'ABC Corp', total: 1250, status: 'pending' },
  { key: '2', id: 'ORD-002', customer: 'XYZ Ltd', total: 3400, status: 'approved' },
  { key: '3', id: 'ORD-003', customer: 'Global Trade', total: 890, status: 'shipped' },
];

export default function Orders() {
  const router = useRouter();
  const [role, setRole] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('erp_access_token');
    if (!token) { router.push('/login'); return; }
    setRole(getRoleFromToken(token));
  }, []);

  const canRead = hasPermission(role, 'orders', 'read') || role === 'Administrator';
  const canApprove = hasPermission(role, 'orders', 'approve') || role === 'Administrator';
  const canWrite = hasPermission(role, 'orders', 'write') || role === 'Administrator';

  if (!canRead) {
    return <Card><h1>403 Forbidden</h1><p>You need orders:read permission.</p></Card>;
  }

  const columns = [
    { title: 'Order ID', dataIndex: 'id' },
    { title: 'Customer', dataIndex: 'customer' },
    { title: 'Total', dataIndex: 'total', render: (v: number) => `$${v}` },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={s === 'pending' ? 'orange' : s === 'approved' ? 'blue' : 'green'}>{s}</Tag> },
    {
      title: 'Actions',
      render: (_: any, record: any) => (
        <>
          {canApprove && record.status === 'pending' && <Button size="small" type="primary" style={{ marginRight: 8 }}>Approve</Button>}
          {canWrite && record.status === 'approved' && <Button size="small">Ship</Button>}
        </>
      ),
    },
  ];

  return (
    <Card title="Orders" extra={canWrite && <Button type="primary">+ New Order</Button>}>
      <Table columns={columns} dataSource={mockOrders} pagination={{ pageSize: 10 }} />
    </Card>
  );
}
