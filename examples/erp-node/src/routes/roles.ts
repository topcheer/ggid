/** Roles routes — CRUD */
import { Router } from 'express';
import { requireAuth, requirePermission } from '../middleware/auth.js';

export const roleRoutes = Router();
roleRoutes.use(requireAuth() as any);

const roles: any[] = [
  { id: '1', name: 'Viewer', permissions: ['dashboard:read', 'inventory:read', 'orders:read'] },
  { id: '2', name: 'Sales', permissions: ['dashboard:read', 'inventory:read', 'orders:read', 'orders:write'] },
  { id: '3', name: 'Manager', permissions: ['dashboard:read', 'inventory:read', 'orders:read', 'orders:read:all', 'orders:write', 'orders:approve', 'users:read'] },
  { id: '4', name: 'Admin', permissions: ['admin'] },
];

roleRoutes.get('/', requirePermission('roles:read') as any, (_req, res) => {
  res.json({ roles });
});

roleRoutes.post('/', requirePermission('roles:write') as any, (req, res) => {
  const { name, permissions } = req.body;
  const role = { id: String(roles.length + 1), name, permissions };
  roles.push(role);
  res.status(201).json(role);
});

roleRoutes.delete('/:id', requirePermission('roles:delete') as any, (req, res) => {
  const idx = roles.findIndex(r => r.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  roles.splice(idx, 1);
  res.json({ status: 'deleted' });
});
