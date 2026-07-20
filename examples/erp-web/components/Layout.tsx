import { Layout, Menu, Tag, Avatar } from 'antd';
import { useRouter } from 'next/router';
import { UserSession, getMenuItems } from '../lib/auth';

const { Sider, Content } = Layout;

export default function AppLayout({ session, children, activeKey }: {
  session: UserSession | null;
  children: React.ReactNode;
  activeKey: string;
}) {
  const router = useRouter();
  const menuItems = getMenuItems(session);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible>
        <div style={{ height: 48, margin: 16, color: '#fff', textAlign: 'center', lineHeight: '48px', fontSize: 18, fontWeight: 'bold' }}>
          GGID ERP
        </div>
        {session && (
          <div style={{ padding: '0 16px 8px' }}>
            {session.roles.map(r => <Tag key={r} color="blue" style={{ marginBottom: 2 }}>{r}</Tag>)}
          </div>
        )}
        <Menu theme="dark" mode="inline" selectedKeys={[activeKey]}
          items={menuItems.map(m => ({ key: m.key, label: m.label, onClick: () => router.push(m.href!) }))}
        />
      </Sider>
      <Content style={{ padding: 24 }}>{children}</Content>
    </Layout>
  );
}
