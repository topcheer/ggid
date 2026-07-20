import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import { UserSession, hasPermission } from '../lib/auth';
import AppLayout from '../components/Layout';
import { Result } from 'antd';

export default function Admin() {
  const router = useRouter();
  const [session, setSession] = useState<UserSession | null>(null);
  useEffect(() => {
    const raw = localStorage.getItem('ggid_session');
    if (!raw) { router.push('/'); return; }
    setSession(JSON.parse(raw));
  }, [router]);
  if (!session) return <div>Loading...</div>;
  if (!hasPermission(session, 'admin')) {
    return <AppLayout session={session} activeKey="admin"><Result status="403" title="403" subTitle="Admin access required." /></AppLayout>;
  }
  return (
    <AppLayout session={session} activeKey="admin">
      <h1>Admin Panel</h1>
      <p>Welcome, administrator {session.display_name}.</p>
      <ul><li>User Management</li><li>System Settings</li><li>Audit Logs</li></ul>
    </AppLayout>
  );
}
