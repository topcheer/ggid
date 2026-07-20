import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { Card, Form, Input, Button, Typography } from 'antd';
import { GGID_URL, CLIENT_ID, REDIRECT_URI, TENANT_ID } from '../lib/auth';

export default function Login() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  const handleLogin = () => {
    setLoading(true);
    const state = `erp_${Date.now()}`;
    const authUrl = `${GGID_URL}/api/v1/oauth/authorize?response_type=code&client_id=${CLIENT_ID}&redirect_uri=${encodeURIComponent(REDIRECT_URI)}&scope=openid+profile+email&state=${state}`;
    window.location.href = authUrl;
  };

  useEffect(() => {
    // If already logged in, redirect to dashboard
    const token = localStorage.getItem('erp_access_token');
    if (token) router.push('/');
  }, []);

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <Card style={{ width: 400, textAlign: 'center' }}>
        <Typography.Title level={3}>Cross-Border ERP</Typography.Title>
        <Typography.Text type="secondary">Sign in with GGID IAM</Typography.Text>
        <div style={{ marginTop: 24 }}>
          <Button type="primary" size="large" block loading={loading} onClick={handleLogin}>
            Login with GGID
          </Button>
        </div>
        <div style={{ marginTop: 16, fontSize: 12, color: '#999' }}>
          Tenant: {TENANT_ID}
        </div>
      </Card>
    </div>
  );
}
