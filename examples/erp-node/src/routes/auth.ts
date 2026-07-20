/** Auth routes — OAuth2 Client Credentials (M2M) + token verification.
 *  Tenant: 00000000-0000-0000-0000-000000000002
 */
import { Router } from 'express';

const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const TENANT = process.env.GGID_TENANT || '00000000-0000-0000-0000-000000000002';
const CLIENT_ID = process.env.ERP_CLIENT_ID || '';
const CLIENT_SECRET = process.env.ERP_CLIENT_SECRET || '';

export const authRoutes = Router();

// M2M token: POST /api/auth/token with client_id + client_secret
authRoutes.post('/token', async (req, res) => {
  const { client_id, client_secret, grant_type } = req.body;
  const cid = client_id || CLIENT_ID;
  const csecret = client_secret || CLIENT_SECRET;

  const r = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': TENANT },
    body: new URLSearchParams({
      grant_type: grant_type || 'client_credentials',
      client_id: cid,
      client_secret: csecret,
    }),
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});

// Verify token
authRoutes.get('/verify', async (req, res) => {
  const token = req.headers.authorization?.replace(/^Bearer\s+/i, '');
  if (!token) return res.status(401).json({ error: { code: 'unauthenticated', message: 'Missing token' } });
  const r = await fetch(`${GGID_URL}/api/v1/auth/verify`, {
    headers: { Authorization: `Bearer ${token}`, 'X-Tenant-ID': TENANT },
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});

// Introspect token (server-side)
authRoutes.post('/introspect', async (req, res) => {
  const { token } = req.body;
  const r = await fetch(`${GGID_URL}/api/v1/oauth/introspect`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-Tenant-ID': TENANT },
    body: new URLSearchParams({ token }),
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});
