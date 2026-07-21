/**
 * Auth routes — OAuth2 Client Credentials (M2M) + token verification.
 * Uses GGID Node SDK for all auth operations.
 * Tenant: 00000000-0000-0000-0000-000000000002
 */
import { Router } from 'express';
import { ggidClient } from '../middleware/auth.js';

const CLIENT_ID = process.env.ERP_CLIENT_ID || '';
const CLIENT_SECRET = process.env.ERP_CLIENT_SECRET || '';
const TENANT = process.env.GGID_TENANT || '00000000-0000-0000-0000-000000000002';

export const authRoutes = Router();

// M2M token: POST /api/auth/token — uses SDK clientCredentials()
authRoutes.post('/token', async (req, res) => {
  const { client_id, client_secret, scope } = req.body;
  try {
    const tokens = await ggidClient.clientCredentials({
      clientId: client_id || CLIENT_ID,
      clientSecret: client_secret || CLIENT_SECRET,
      scope: scope || undefined,
      tenantId: TENANT,
    });
    res.json(tokens);
  } catch (e: any) {
    res.status(401).json({
      error: { code: 'token_exchange_failed', message: e.message || 'Client credentials authentication failed' },
    });
  }
});

// Verify token — uses SDK verifyToken()
authRoutes.get('/verify', async (req, res) => {
  const token = req.headers.authorization?.replace(/^Bearer\s+/i, '');
  if (!token) return res.status(401).json({ error: { code: 'unauthenticated', message: 'Missing token' } });
  try {
    const claims = await ggidClient.verifyToken(token);
    res.json(claims);
  } catch (e: any) {
    res.status(401).json({ error: { code: 'unauthenticated', message: 'Invalid token' } });
  }
});

// Introspect token — uses SDK introspectToken() if available, fallback to verifyToken
authRoutes.post('/introspect', async (req, res) => {
  const { token } = req.body;
  if (!token) return res.status(400).json({ error: 'missing token' });
  try {
    const claims = await ggidClient.verifyToken(token);
    res.json({
      active: true,
      sub: claims.sub,
      username: (claims as any).username,
      email: (claims as any).email,
      permissions: (claims as any).permissions || (claims as any).scope?.split(' ') || [],
      exp: claims.exp,
    });
  } catch {
    res.json({ active: false });
  }
});