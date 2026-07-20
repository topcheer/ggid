'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Table, Card, Typography, Tag, Space, Result } from 'antd';
import { PlusOutlined, CheckOutlined } from '@ant-design/icons';
import { ERPLayout } from '@/components/ERPLayout';
import { ERPUser, getUser, hasPermission } from '@/lib/auth';

export default function OrdersPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);

  useEffect(() => {
    const u = getUser();
    if (!u) { router.push('/login'); return; }
    setUser(u);
  }, [router]);

  if (!user) return null;
  if (!hasPermission(user, 'orders:read')) {
    return (
      <ERPLayout user={user}>
        <Result status="403" title="403" subTitle="您没有权限访问订单管理" />
      </ERPLayout>
    );
  }

  const canWrite = hasPermission(user, 'orders:write');
  const canApprove = hasPermission(user, 'orders:approve');

  const columns = [
    { title: '订单号', dataIndex: 'id', key: 'id' },
    { title: '客户', dataIndex: 'customer', key: 'customer' },
    { title: '金额', dataIndex: 'amount', key: 'amount' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => {
      const colors: Record<string, string> = { pending: 'orange', approved: 'green', shipped: 'blue' };
      return <Tag color={colors[s] || 'default'}>{s}</Tag>;
    }},
    ...(canApprove ? [{ title: '操作', key: 'action', render: () => (
      <Space>
        <Button type="primary" size="small" icon={<CheckOutlined />}>审批</Button>
      </Space>
    )}] : []),
  ];

  return (
    <ERPLayout user={user}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={3}>订单管理</Typography.Title>
        {canWrite && <Button type="primary" icon={<PlusOutlined />}>新建订单</Button>}
      </div>
      <Card>
        <Table columns={columns} dataSource={[]} locale={{ emptyText: '暂无订单数据' }} />
      </Card>
    </ERPLayout>
  );
}
