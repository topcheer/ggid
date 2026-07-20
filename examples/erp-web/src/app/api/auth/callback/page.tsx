'use client';
import { useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

export default function AuthCallback() {
  const router = useRouter();
  const params = useSearchParams();

  useEffect(() => {
    const code = params.get('code');
    if (!code) { router.push('/login'); return; }

    // Exchange code for token via server-side API route
    fetch('/api/auth/exchange', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    })
      .then(res => res.json())
      .then(data => {
        if (data.access_token) {
          localStorage.setItem('erp_access_token', data.access_token);
          router.push('/dashboard');
        } else {
          router.push('/login');
        }
      })
      .catch(() => router.push('/login'));
  }, [router, params]);

  return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>Processing login...</div>;
}
