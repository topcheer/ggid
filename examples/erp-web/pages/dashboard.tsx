import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import { UserSession, hasPermission } from '../lib/auth';
import AppLayout from '../components/Layout';
import { Card, Statistic, Row, Col, Tag } from 'antd';

export default function Dashboard() {
  const router = useRouter();
  const [session, setSession] = useState<UserSession | null>(null);
  useEffect(() => {
    const raw = localStorage.getItem('ggid_session');
    if (!raw) { router.push('/'); return; }
    setSession(JSON.parse(raw));
  }, [router]);
  if (!session) return <div>Loading...</div>;
  const perms = ['inventory:read', 'inventory:write', 'orders:read', 'orders:write', 'orders:approve', 'reports:read', 'admin'];
  return (
    <AppLayout session={session} activeKey="dashboard">
      <h1>Dashboard</h1>
      <p>Welcome, <strong>{session.display_name}</strong></p>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}><Card><Statistic title="Orders" value={142} /></Card></Col>
        <Col span={6}><Card><Statistic title="Revenue" value={48200} prefix="$" /></Card></Col>
        <Col span={6}><Card><Statistic title="Products" value={38} /></Card></Col>
        <Col span={6}><Card><Statistic title="Low Stock" value={3} valueStyle={{ color: '#cf1322' }} /></Card></Col>
      </Row>
      <h3>Your Permissions</h3>
      <div>{perms.map(p => (
        <Tag key={p} color={hasPermission(session, p) ? 'green' : 'red'} style={{ margin: 4 }}>
          {hasPermission(session, p) ? 'YES' : 'NO'} {p}
        </Tag>
      ))}</div>
    </AppLayout>
  );
}
