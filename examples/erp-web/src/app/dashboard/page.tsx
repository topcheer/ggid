'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Col, Row, Statistic, Typography, Tag, List } from 'antd';
import { ERPLayout } from '@/components/ERPLayout';
import { ERPUser, getUser, hasPermission } from '@/lib/auth';

export default function DashboardPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);

  useEffect(() => {
    const u = getUser();
    if (!u) { router.push('/login'); return; }
    setUser(u);
  }, [router]);

  if (!user) return null;

  return (
    <ERPLayout user={user}>
      <Typography.Title level={3}>Dashboard</Typography.Title>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}><Card><Statistic title="Total Orders" value={1247} /></Card></Col>
        <Col span={6}><Card><Statistic title="Pending" value={23} /></Card></Col>
        <Col span={6}><Card><Statistic title="Products" value={458} /></Card></Col>
        <Col span={6}><Card><Statistic title="Revenue" value="$128,540" /></Card></Col>
      </Row>
      <Card title="Your Permissions">
        <List
          size="small"
          dataSource={user.permissions}
          renderItem={(perm) => (
            <List.Item>
              <Tag color={perm.includes('write') || perm.includes('delete') || perm.includes('approve') ? 'green' : 'blue'}>{perm}</Tag>
            </List.Item>
          )}
        />
      </Card>
    </ERPLayout>
  );
}
