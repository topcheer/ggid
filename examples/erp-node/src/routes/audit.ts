/** Audit routes — view audit log (all roles can read) */
import { Router } from 'express';
import { requireAuth, requirePermission } from '../middleware/auth.js';

export const auditRoutes = Router();
auditRoutes.use(requireAuth() as any);

const events: any[] = [
  { id: '1', event_type: 'user.login', actor: 'admin', action: 'login', result: 'success', timestamp: new Date().toISOString() },
  { id: '2', event_type: 'order.create', actor: 'sales', action: 'create', result: 'success', timestamp: new Date().toISOString() },
  { id: '3', event_type: 'inventory.update', actor: 'warehouse', action: 'update', result: 'success', timestamp: new Date().toISOString() },
];

auditRoutes.get('/', requirePermission('audit:read') as any, (req, res) => {
  res.json({ events, total: events.length });
});
