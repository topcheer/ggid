/**
 * Cross-Board ERP Demo — Node.js Backend
 * Uses GGID Node SDK for authentication and fine-grained permissions.
 *
 * Run: GGID_URL=https://ggid.example.com PORT=3200 npx tsx src/index.ts
 */
import express from 'express';
import cors from 'cors';
import { authRoutes } from './routes/auth.js';
import { userRoutes } from './routes/users.js';
import { roleRoutes } from './routes/roles.js';
import { orgRoutes } from './routes/orgs.js';
import { inventoryRoutes } from './routes/inventory.js';
import { orderRoutes } from './routes/orders.js';
import { auditRoutes } from './routes/audit.js';

const PORT = parseInt(process.env.PORT || '3200');

const app = express();
app.use(cors());
app.use(express.json());

// Routes
app.use('/api/auth', authRoutes);
app.use('/api/users', userRoutes);
app.use('/api/roles', roleRoutes);
app.use('/api/orgs', orgRoutes);
app.use('/api/inventory', inventoryRoutes);
app.use('/api/orders', orderRoutes);
app.use('/api/audit', auditRoutes);

// Health
app.get('/health', (_req, res) => res.json({ status: 'ok' }));

app.listen(PORT, () => console.log(`ERP Node Backend on http://localhost:${PORT}`));
