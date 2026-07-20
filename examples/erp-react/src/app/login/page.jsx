'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { getUser, buildAuthUrl } from '../../lib/auth';

export default function LoginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  useEffect(() => { if (getUser()) router.push('/dashboard'); }, [router]);

  const handleLogin = async () => {
    setLoading(true);
    const authUrl = await buildAuthUrl();
    window.location.href = authUrl;
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', fontFamily: 'system-ui' }}>
      <div style={{ width: 360, padding: 32, background: '#fff', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.1)' }}>
        <h2 style={{ margin: '0 0 8px' }}>Cross-Board ERP</h2>
        <p style={{ color: '#999', fontSize: 14 }}>Sign in with GGID (OAuth2 + PKCE)</p>
        <button
          onClick={handleLogin}
          disabled={loading}
          style={{
            width: '100%', padding: 12, fontSize: 16, background: '#1890ff',
            color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer', marginTop: 16,
            opacity: loading ? 0.7 : 1,
          }}
        >
          {loading ? 'Redirecting...' : 'Login with GGID'}
        </button>
      </div>
    </div>
  );
}