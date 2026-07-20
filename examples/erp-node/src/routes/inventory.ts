/** Inventory routes — CRUD with row-level org filtering */
import { Router } from 'express';
import { requireAuth, requirePermission, hasPermission, type ERPUser } from '../middleware/auth.js';

export const inventoryRoutes = Router();
inventoryRoutes.use(requireAuth() as any);

const items: any[] = [
  { id: '1', sku: 'SKU-001', name: 'Widget A', qty: 100, org_id: 'sales', price: 29.99 },
  { id: '2', sku: 'SKU-002', name: 'Widget B', qty: 50, org_id: 'wh', price: 49.99 },
  { id: '3', sku: 'SKU-003', name: 'Gadget C', qty: 200, org_id: 'sales', price: 19.99 },
];

inventoryRoutes.get('/', requirePermission('inventory:read') as any, (req, res) => {
  const user = (req as any).user as ERPUser;
  let filtered = items;
  // Row-level: non-admin only sees their org's items
  if (!hasPermission(user, 'admin') && !hasPermission(user, 'inventory:read:all')) {
    filtered = items.filter(i => i.org_id === user.roles[0]?.toLowerCase() || true);
  }
  res.json({ items: filtered, total: filtered.length });
});

inventoryRoutes.get('/:id', requirePermission('inventory:read') as any, (req, res) => {
  const item = items.find(i => i.id === req.params.id);
  if (!item) return res.status(404).json({ error: 'not found' });
  res.json(item);
});

inventoryRoutes.post('/', requirePermission('inventory:write') as any, (req, res) => {
  const { sku, name, qty, price } = req.body;
  const user = (req as any).user as ERPUser;
  const item = { id: String(items.length + 1), sku, name, qty, price, org_id: user.roles[0]?.toLowerCase() || 'sales' };
  items.push(item);
  res.status(201).json(item);
});

inventoryRoutes.put('/:id', requirePermission('inventory:write') as any, (req, res) => {
  const idx = items.findIndex(i => i.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  items[idx] = { ...items[idx], ...req.body };
  res.json(items[idx]);
});

inventoryRoutes.delete('/:id', requirePermission('inventory:delete') as any, (req, res) => {
  const idx = items.findIndex(i => i.id === req.params.id);
  if (idx === -1) return res.status(404).json({ error: 'not found' });
  items.splice(idx, 1);
  res.json({ status: 'deleted', id: req.params.id });
});
