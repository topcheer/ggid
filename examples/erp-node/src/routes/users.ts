/** Users routes — CRUD with permission checks */
import { Router } from 'express';
import { requireAuth, requirePermission, hasPermission, type ERPUser } from '../middleware/auth.js';

export const userRoutes = Router();
userRoutes.use(requireAuth() as any);

// In-memory store
const users: any[] = [
  { id: '1', username: 'viewer', email: 'viewer@erp.com', role: 'Viewer', status: 'active' },
  { id: '2', username: 'sales', email: 'sales@erp.com', role: 'Sales', status: 'active' },
  { id: '3', username: 'manager', email: 'manager@erp.com', role: 'Manager', status: 'active' },
];

userRoutes.get('/', requirePermission('users:read') as any, (req, res) => {
  res.json({ users, total: users.length });
});

userRoutes.get('/:id', requirePermission('users:read') as any, (req, res) => {
  const user = users.find(u => u.id === req.params.id);
  if (!user) return res.status(404).json({ error: 'not found' });
  res.json(user);
});

userRoutes.post('/', requirePermission('users:write') as any, (req, res) => {
  const { username, email, role } = req.body;
  const user = { id: String(users.length + 1), username, email, role, status: 'active' };
  users.push(user);
  res.status(201).json(user);
});

userRoutes.put('/:id', requirePermission('users:write') as any, (req, res) => {
  const idx = users.findIndex(u => u.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  users[idx] = { ...users[idx], ...req.body };
  res.json(users[idx]);
});

userRoutes.delete('/:id', requirePermission('users:delete') as any, (req, res) => {
  const idx = users.findIndex(u => u.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  users.splice(idx, 1);
  res.json({ status: 'deleted', id: req.params.id });
});
