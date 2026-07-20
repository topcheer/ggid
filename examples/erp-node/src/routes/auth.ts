/** Auth routes — login, refresh, verify, logout */
import { Router } from 'express';

const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const TENANT = process.env.GGID_TENANT || '00000000-0000-0000-0000-000000000001';

export const authRoutes = Router();

authRoutes.post('/login', async (req, res) => {
  const { username, password } = req.body;
  const r = await fetch(`${GGID_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': TENANT },
    body: JSON.stringify({ username, password }),
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});

authRoutes.post('/refresh', async (req, res) => {
  const { refresh_token } = req.body;
  const r = await fetch(`${GGID_URL}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token }),
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});

authRoutes.get('/verify', async (req, res) => {
  const token = req.headers.authorization?.replace(/^Bearer\s+/i, '');
  if (!token) return res.status(401).json({ error: 'missing token' });
  const r = await fetch(`${GGID_URL}/api/v1/auth/verify`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = await r.json();
  if (!r.ok) return res.status(r.status).json(data);
  res.json(data);
});

authRoutes.post('/logout', async (req, res) => {
  res.json({ status: 'ok' });
});
