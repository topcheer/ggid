'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Table, Card, Typography, Tag, Space, Result } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { ERPLayout } from '@/components/ERPLayout';
import { ERPUser, getUser, hasPermission } from '@/lib/auth';

export default function InventoryPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);

  useEffect(() => {
    const u = getUser();
    if (!u) { router.push('/login'); return; }
    setUser(u);
  }, [router]);

  if (!user) return null;
  if (!hasPermission(user, 'inventory:read')) {
    return (
      <ERPLayout user={user}>
        <Result status="403" title="403" subTitle="您没有权限访问库存管理" />
      </ERPLayout>
    );
  }

  const canWrite = hasPermission(user, 'inventory:write');
  const canDelete = hasPermission(user, 'inventory:delete');

  const columns = [
    { title: 'SKU', dataIndex: 'sku', key: 'sku' },
    { title: '产品名称', dataIndex: 'name', key: 'name' },
    { title: '数量', dataIndex: 'qty', key: 'qty' },
    { title: '仓库', dataIndex: 'warehouse', key: 'warehouse' },
    ...(canDelete ? [{ title: '操作', key: 'action', render: () => <Button danger icon={<DeleteOutlined />} size="small" /> }] : []),
  ];

  return (
    <ERPLayout user={user}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={3}>库存管理</Typography.Title>
        {canWrite && <Button type="primary" icon={<PlusOutlined />} onClick={() => router.push('/inventory/new')}>新建库存</Button>}
      </div>
      <Card>
        <Table columns={columns} dataSource={[]} locale={{ emptyText: '暂无库存数据' }} />
      </Card>
    </ERPLayout>
  );
}
