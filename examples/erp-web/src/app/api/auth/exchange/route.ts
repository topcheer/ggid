import { NextRequest, NextResponse } from 'next/server';
import { GGID_URL, CLIENT_ID, CLIENT_SECRET, REDIRECT_URI } from '@/lib/auth';

export async function POST(req: NextRequest) {
  const { code } = await req.json();

  const res = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
    }),
  });

  if (!res.ok) {
    return NextResponse.json({ error: 'token_exchange_failed' }, { status: 500 });
  }

  const tokens = await res.json();
  return NextResponse.json(tokens);
}
