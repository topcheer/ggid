/** Orders routes — CRUD with approval + row-level org filtering */
import { Router } from 'express';
import { requireAuth, requirePermission, hasPermission, type ERPUser } from '../middleware/auth.js';

export const orderRoutes = Router();
orderRoutes.use(requireAuth() as any);

const orders: any[] = [
  { id: 'ORD-001', customer: 'Alice Corp', amount: 1200, status: 'pending', org_id: 'sales', group_id: 'team-a' },
  { id: 'ORD-002', customer: 'Bob Inc', amount: 3400, status: 'approved', org_id: 'sales', group_id: 'team-b' },
];

orderRoutes.get('/', requirePermission('orders:read') as any, (req, res) => {
  const user = (req as any).user as ERPUser;
  let filtered = orders;
  // Row-level: Manager sees all, others see their group
  if (!hasPermission(user, 'orders:read:all') && !hasPermission(user, 'admin')) {
    filtered = orders.filter(o => o.org_id === 'sales'); // simplified
  }
  res.json({ orders: filtered, total: filtered.length });
});

orderRoutes.get('/:id', requirePermission('orders:read') as any, (req, res) => {
  const order = orders.find(o => o.id === req.params.id);
  if (!order) return res.status(404).json({ error: 'not found' });
  res.json(order);
});

orderRoutes.post('/', requirePermission('orders:write') as any, (req, res) => {
  const { customer, amount } = req.body;
  const user = (req as any).user as ERPUser;
  const order = { id: `ORD-${String(orders.length + 1).padStart(3, '0')}`, customer, amount, status: 'pending', org_id: 'sales', group_id: 'team-a' };
  orders.push(order);
  res.status(201).json(order);
});

orderRoutes.post('/:id/approve', requirePermission('orders:approve') as any, (req, res) => {
  const order = orders.find(o => o.id === req.params.id);
  if (!order) return res.status(404).json({ error: 'not found' });
  if (order.status !== 'pending') return res.status(400).json({ error: 'order not pending' });
  order.status = 'approved';
  res.json(order);
});

orderRoutes.put('/:id', requirePermission('orders:write') as any, (req, res) => {
  const idx = orders.findIndex(o => o.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  orders[idx] = { ...orders[idx], ...req.body };
  res.json(orders[idx]);
});

orderRoutes.delete('/:id', requirePermission('orders:delete') as any, (req, res) => {
  const idx = orders.findIndex(o => o.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  orders.splice(idx, 1);
  res.json({ status: 'deleted' });
});
