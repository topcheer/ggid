import { useEffect } from 'react';
import { useRouter } from 'next/router';
import { GGID_URL, CLIENT_ID, CLIENT_SECRET, REDIRECT_URI, decodeJWT, UserSession } from '../lib/auth';

export default function Callback() {
  const router = useRouter();
  useEffect(() => {
    const code = new URLSearchParams(window.location.search).get('code');
    if (!code) { router.push('/'); return; }
    fetch(`${GGID_URL}/api/v1/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ grant_type: 'authorization_code', code, client_id: CLIENT_ID, client_secret: CLIENT_SECRET, redirect_uri: REDIRECT_URI }),
    })
      .then(r => r.json())
      .then((data: any) => {
        if (!data.access_token) { router.push('/?error=token_failed'); return; }
        const claims = decodeJWT(data.access_token) || {};
        const session: UserSession = {
          access_token: data.access_token,
          username: claims.sub || 'unknown', email: claims.email || '',
          display_name: claims.name || claims.sub || 'User',
          scopes: (claims.scopes || []) as string[], roles: (claims.scopes || []) as string[],
        };
        localStorage.setItem('ggid_session', JSON.stringify(session));
        router.push('/dashboard');
      })
      .catch(() => router.push('/?error=exchange_failed'));
  }, [router]);
  return <div style={{ padding: 40, textAlign: 'center' }}>Processing authentication...</div>;
}
