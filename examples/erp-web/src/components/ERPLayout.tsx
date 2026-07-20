'use client';
import React from 'react';
import { Layout, Menu, Avatar, Dropdown, Tag, theme } from 'antd';
import { DashboardOutlined, ShoppingOutlined, InboxOutlined, SettingOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';
import { ERPUser, hasPermission, logout } from '@/lib/auth';

const { Sider, Header, Content } = Layout;

export function ERPLayout({ user, children }: { user: ERPUser; children: React.ReactNode }) {
  const roleColors: Record<string, string> = {
    sales_manager: 'blue',
    warehouse_manager: 'green',
    finance_officer: 'gold',
    admin: 'red',
  };

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
  ];

  if (hasPermission(user, 'orders:read'))
    menuItems.push({ key: '/orders', icon: <ShoppingOutlined />, label: '订单管理' });
  if (hasPermission(user, 'inventory:read'))
    menuItems.push({ key: '/inventory', icon: <InboxOutlined />, label: '库存管理' });
  if (hasPermission(user, 'admin'))
    menuItems.push({ key: '/admin', icon: <SettingOutlined />, label: '系统管理' });

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider breakpoint="lg" collapsedWidth="0">
        <div style={{ padding: '16px 24px', color: '#fff', fontSize: '18px', fontWeight: 'bold' }}>
          跨境ERP系统
        </div>
        <div style={{ padding: '0 16px 16px' }}>
          {user.roles.map((role: string) => (
            <Tag key={role} color={roleColors[role] || 'default'} style={{ marginBottom: 4 }}>
              {role.replace(/_/g, ' ').toUpperCase()}
            </Tag>
          ))}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          items={menuItems}
          onClick={({ key }) => window.location.href = key}
        />
      </Sider>
      <Layout>
        <Header style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
          <Dropdown menu={{
            items: [
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: logout },
            ],
          }}>
            <span style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
              <Avatar icon={<UserOutlined />} />
              <span style={{ color: '#fff' }}>{user.displayName}</span>
            </span>
          </Dropdown>
        </Header>
        <Content style={{ padding: '24px' }}>
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
