'use client';
import { useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { exchangeCodeForToken } from '../../lib/auth';

export default function CallbackPage() {
  const router = useRouter();
  const params = useSearchParams();

  useEffect(() => {
    const code = params.get('code');
    if (!code) { router.push('/login'); return; }

    exchangeCodeForToken(code).then(token => {
      if (token) router.push('/dashboard');
      else router.push('/login');
    }).catch(() => router.push('/login'));
  }, [router, params]);

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', fontFamily: 'system-ui' }}>
      <p>Processing login...</p>
    </div>
  );
}