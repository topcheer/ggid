/**
 * GGID Node.js SDK Quickstart — 5-minute JWT authentication integration.
 *
 * Shows how to:
 * 1. Login and get a JWT token
 * 2. Protect Express routes with GGID middleware
 * 3. Access user info from the JWT in your handlers
 *
 * Prerequisites:
 *   - GGID running (cd deploy && docker compose up -d)
 *   - Node.js 18+
 *
 * Run:
 *   npm install express
 *   npx tsx index.ts
 *
 * Test:
 *   curl http://localhost:9090/public           → 200 (no auth needed)
 *   curl http://localhost:9090/protected        → 401 (missing token)
 *   curl -H "Authorization: Bearer <token>" http://localhost:9090/protected → 200
 */
import express from 'express';
import { GGIDClient, expressAuth, getClaims } from '@ggid/node';

const app = express();
app.use(express.json());

const GATEWAY_URL = process.env.GGID_GATEWAY_URL || 'http://localhost:8080';
const TENANT_ID = process.env.GGID_TENANT_ID || '00000000-0000-0000-0000-000000000001';

// Step 1: Create client and login
const client = new GGIDClient({
  gatewayUrl: GATEWAY_URL,
  tenantId: TENANT_ID,
});

async function main() {
  const tokens = await client.login({
    username: 'admin',
    password: 'Admin@123456',
  });
  console.log(`Login OK — token length: ${tokens.access_token.length}`);

  // Step 2: Public route (no auth)
  app.get('/public', (_req, res) => {
    res.json({ message: 'public endpoint, no auth needed' });
  });

  // Step 3: Protect all /api/* routes with GGID middleware
  app.use('/api', expressAuth({
    gatewayUrl: GATEWAY_URL,
    skipPaths: ['/api/health'],
  }));

  // Protected route — user info available via getClaims()
  app.get('/api/me', (req, res) => {
    const claims = getClaims(req);
    res.json({
      message: 'authenticated!',
      user: claims?.sub,
      email: claims?.email,
      roles: claims?.roles,
    });
  });

  app.get('/api/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  app.listen(9090, () => {
    console.log('Quickstart server running on :9090');
    console.log('  Public:    http://localhost:9090/public');
    console.log('  Protected: http://localhost:9090/api/me');
    console.log(`  Token:      ${tokens.access_token.slice(0, 50)}...`);
  });
}

main().catch(console.error);
