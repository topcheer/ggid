import { useEffect, useState } from 'react';
import { Layout, Menu, Tag, Card, Descriptions, Button } from 'antd';
import { DashboardOutlined, ShoppingCartOutlined, ShopOutlined, BarChartOutlined, SettingOutlined, LogoutOutlined } from '@ant-design/icons';
import { useRouter } from 'next/router';
import { getRoleFromToken, getPermissions, hasPermission } from '../lib/auth';

const { Header, Sider, Content } = Layout;

export default function Home() {
  const router = useRouter();
  const [role, setRole] = useState<string>('');
  const [perms, setPerms] = useState<string[]>([]);

  useEffect(() => {
    const token = localStorage.getItem('erp_access_token');
    if (!token) { router.push('/login'); return; }
    const r = getRoleFromToken(token);
    setRole(r);
    setPerms(getPermissions(r));
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('erp_access_token');
    router.push('/login');
  };

  const canSeeInventory = hasPermission(role, 'inventory', 'read') || role === 'Administrator';
  const canSeeOrders = hasPermission(role, 'orders', 'read') || role === 'Administrator';
  const canSeeReports = hasPermission(role, 'reports', 'read') || role === 'Administrator';
  const isAdmin = role === 'Administrator';

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: 'Dashboard' },
    ...(canSeeOrders ? [{ key: '/orders', icon: <ShoppingCartOutlined />, label: 'Orders' }] : []),
    ...(canSeeInventory ? [{ key: '/inventory', icon: <ShopOutlined />, label: 'Inventory' }] : []),
    ...(canSeeReports ? [{ key: '/reports', icon: <BarChartOutlined />, label: 'Reports' }] : []),
    ...(isAdmin ? [{ key: '/admin', icon: <SettingOutlined />, label: 'Admin' }] : []),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible>
        <div style={{ height: 48, margin: 16, textAlign: 'center', color: '#fff', fontSize: 18, fontWeight: 'bold' }}>
          ERP Demo
        </div>
        {role && (
          <div style={{ textAlign: 'center', marginBottom: 16 }}>
            <Tag color="blue" style={{ fontSize: 14 }}>{role}</Tag>
          </div>
        )}
        <Menu theme="dark" mode="inline" items={menuItems} onClick={(e) => router.push(e.key)} selectedKeys={[router.pathname]} />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ margin: 0 }}>Cross-Border ERP</h2>
          <Button icon={<LogoutOutlined />} onClick={handleLogout}>Logout</Button>
        </Header>
        <Content style={{ margin: 24 }}>
          <Card>
            <Descriptions title="Dashboard" bordered>
              <Descriptions.Item label="Role">{role || 'Loading...'}</Descriptions.Item>
              <Descriptions.Item label="Permissions">
                {perms.map(p => <Tag key={p} color="geekblue">{p}</Tag>)}
              </Descriptions.Item>
            </Descriptions>
            <Card type="inner" title="Permission Status" style={{ marginTop: 16 }}>
              <p>Inventory Read: <Tag color={hasPermission(role, 'inventory', 'read') ? 'green' : 'red'}>{String(hasPermission(role, 'inventory', 'read'))}</Tag></p>
              <p>Inventory Write: <Tag color={hasPermission(role, 'inventory', 'write') ? 'green' : 'red'}>{String(hasPermission(role, 'inventory', 'write'))}</Tag></p>
              <p>Orders Read: <Tag color={hasPermission(role, 'orders', 'read') ? 'green' : 'red'}>{String(hasPermission(role, 'orders', 'read'))}</Tag></p>
              <p>Orders Approve: <Tag color={hasPermission(role, 'orders', 'approve') ? 'green' : 'red'}>{String(hasPermission(role, 'orders', 'approve'))}</Tag></p>
              <p>Reports Read: <Tag color={hasPermission(role, 'reports', 'read') ? 'green' : 'red'}>{String(hasPermission(role, 'reports', 'read'))}</Tag></p>
            </Card>
          </Card>
        </Content>
      </Layout>
    </Layout>
  );
}
