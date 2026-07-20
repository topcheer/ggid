import { useEffect } from 'react';
import { useRouter } from 'next/router';
import { Spin, message } from 'antd';
import { GGID_URL, CLIENT_ID, CLIENT_SECRET, REDIRECT_URI, TENANT_ID } from '../lib/auth';

export default function Callback() {
  const router = useRouter();

  useEffect(() => {
    const code = new URLSearchParams(window.location.search).get('code');
    if (!code) { router.push('/login'); return; }

    // Exchange code for token via server-side (client-side for demo)
    fetch(`${GGID_URL}/api/v1/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': TENANT_ID },
      body: new URLSearchParams({
        grant_type: 'authorization_code',
        code,
        redirect_uri: REDIRECT_URI,
        client_id: CLIENT_ID,
        client_secret: CLIENT_SECRET,
      }),
    })
      .then(r => r.json())
      .then(data => {
        if (data.access_token) {
          localStorage.setItem('erp_access_token', data.access_token);
          router.push('/');
        } else {
          message.error('Login failed');
          router.push('/login');
        }
      })
      .catch(() => { router.push('/login'); });
  }, []);

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
      <Spin size="large" tip="Completing login..." />
    </div>
  );
}
