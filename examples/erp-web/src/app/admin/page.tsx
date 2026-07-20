'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Result, Card, Typography, Descriptions } from 'antd';
import { ERPLayout } from '@/components/ERPLayout';
import { ERPUser, getUser, hasPermission } from '@/lib/auth';

export default function AdminPage() {
  const router = useRouter();
  const [user, setUser] = useState<ERPUser | null>(null);

  useEffect(() => {
    const u = getUser();
    if (!u) { router.push('/login'); return; }
    setUser(u);
  }, [router]);

  if (!user) return null;
  if (!hasPermission(user, 'admin')) {
    return (
      <ERPLayout user={user}>
        <Result status="403" title="403" subTitle="您没有管理员权限" />
      </ERPLayout>
    );
  }

  return (
    <ERPLayout user={user}>
      <Typography.Title level={3}>系统管理</Typography.Title>
      <Card>
        <Descriptions title="系统信息" column={1}>
          <Descriptions.Item label="版本">ERP Demo v1.0</Descriptions.Item>
          <Descriptions.Item label="认证">GGID IAM Suite</Descriptions.Item>
          <Descriptions.Item label="当前用户">{user.username}</Descriptions.Item>
        </Descriptions>
      </Card>
    </ERPLayout>
  );
}
