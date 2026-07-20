'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Card, Typography } from 'antd';
import { GGID_URL, CLIENT_ID, REDIRECT_URI, getUser } from '@/lib/auth';

export default function LoginPage() {
  const router = useRouter();

  useEffect(() => {
    const user = getUser();
    if (user) router.push('/dashboard');
  }, [router]);

  const handleLogin = () => {
    const authUrl = `${GGID_URL}/api/v1/oauth/authorize?` + new URLSearchParams({
      response_type: 'code',
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: 'openid profile email',
      state: 'erp-demo',
    }).toString();
    window.location.href = authUrl;
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', background: '#f0f2f5' }}>
      <Card style={{ width: 400, textAlign: 'center' }}>
        <Typography.Title level={2}>跨境ERP系统</Typography.Title>
        <Typography.Paragraph type="secondary">使用 GGID 统一身份认证登录</Typography.Paragraph>
        <Button type="primary" size="large" block onClick={handleLogin}>
          通过 GGID 登录
        </Button>
      </Card>
    </div>
  );
}
