/** Orgs routes — CRUD */
import { Router } from 'express';
import { requireAuth, requirePermission } from '../middleware/auth.js';

export const orgRoutes = Router();
orgRoutes.use(requireAuth() as any);

const orgs: any[] = [
  { id: '1', name: 'Sales Dept', code: 'sales', parent_id: null },
  { id: '2', name: 'Warehouse', code: 'wh', parent_id: null },
  { id: '3', name: 'Finance', code: 'fin', parent_id: null },
];

orgRoutes.get('/', requirePermission('orgs:read') as any, (_req, res) => res.json({ orgs }));
orgRoutes.post('/', requirePermission('orgs:write') as any, (req, res) => {
  const { name, code, parent_id } = req.body;
  const org = { id: String(orgs.length + 1), name, code, parent_id };
  orgs.push(org);
  res.status(201).json(org);
});
orgRoutes.delete('/:id', requirePermission('orgs:delete') as any, (req, res) => {
  const idx = orgs.findIndex(o => o.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  orgs.splice(idx, 1);
  res.json({ status: 'deleted' });
});
