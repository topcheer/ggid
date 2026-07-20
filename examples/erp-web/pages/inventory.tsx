import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { Card, Table, Button, Tag } from 'antd';
import { getRoleFromToken, hasPermission } from '../lib/auth';

const mockInventory = [
  { key: '1', sku: 'ERP-001', name: 'Wireless Mouse', qty: 120, location: 'WH-A' },
  { key: '2', sku: 'ERP-002', name: 'USB Hub', qty: 45, location: 'WH-B' },
  { key: '3', sku: 'ERP-003', name: 'HDMI Cable', qty: 200, location: 'WH-A' },
];

export default function Inventory() {
  const router = useRouter();
  const [role, setRole] = useState('');

  useEffect(() => {
    const token = localStorage.getItem('erp_access_token');
    if (!token) { router.push('/login'); return; }
    setRole(getRoleFromToken(token));
  }, []);

  const canRead = hasPermission(role, 'inventory', 'read') || role === 'Administrator';
  const canWrite = hasPermission(role, 'inventory', 'write') || role === 'Administrator';
  const canDelete = hasPermission(role, 'inventory', 'delete') || role === 'Administrator';

  if (!canRead) {
    return <Card><h1>403 Forbidden</h1><p>You need inventory:read permission.</p><p>Current role: {role}</p></Card>;
  }

  const columns = [
    { title: 'SKU', dataIndex: 'sku' },
    { title: 'Name', dataIndex: 'name' },
    { title: 'Quantity', dataIndex: 'qty' },
    { title: 'Location', dataIndex: 'location' },
    {
      title: 'Actions',
      render: () => (
        <>
          {canDelete && <Button danger size="small">Delete</Button>}
        </>
      ),
    },
  ];

  return (
    <Card title="Inventory Management" extra={canWrite && <Button type="primary">+ New Item</Button>}>
      <Table columns={columns} dataSource={mockInventory} pagination={{ pageSize: 10 }} />
      {!canWrite && <Tag color="orange">Read-only mode (no inventory:write)</Tag>}
    </Card>
  );
}
