# Express.js Integration Guide

> Add GGID authentication to an Express.js app using the Node SDK.

---

## Install

```bash
npm install @ggid/node express
```

## Minimal Setup

```javascript
const express = require('express');
const { expressAuth, getClaims } = require('@ggid/node');

const app = express();

// Protect all /api/* routes
app.use('/api', expressAuth({
  jwksUrl: process.env.JWKS_URL || 'http://localhost:8080/.well-known/jwks.json',
  issuer: process.env.GGID_URL || 'http://localhost:8080',
}));

app.get('/api/profile', (req, res) => {
  const claims = getClaims(req);
  res.json({
    userID: claims.sub,
    tenantID: claims.tenant_id,
    scopes: claims.scope?.split(' ') || [],
  });
});

app.listen(3000, () => console.log('Server on :3000'));
```

## Role-Based Authorization

```javascript
const { requireRole, requirePermission } = require('@ggid/node');

// Require admin role (local JWT claim check)
app.delete('/api/users/:id', requireRole('admin'), async (req, res) => {
  // ...
});

// Check permission via Policy Engine
app.get('/api/users', requirePermission(
  { gatewayUrl: 'http://localhost:8080' },
  'users', 'read',
), async (req, res) => {
  // ...
});
```

## Tenant-Aware Database Queries

```javascript
app.get('/api/items', (req, res) => {
  const claims = getClaims(req);
  const tenantID = claims.tenant_id;
  db.query('SELECT * FROM items WHERE tenant_id = $1', [tenantID]);
});
```

## Optional Auth (Public + Protected Routes)

```javascript
const { expressAuth } = require('@ggid/node');

// Public routes (no middleware)
app.get('/health', (req, res) => res.json({ ok: true }));
app.post('/api/auth/login', loginHandler);

// Protected routes
app.use('/api', expressAuth({
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
}));
```

## Using the GGID Client

```javascript
const { GGIDClient } = require('@ggid/node');

app.get('/api/users/:id', requireRole('admin'), async (req, res) => {
  const client = new GGIDClient({
    gatewayUrl: process.env.GGID_URL,
    apiKey: process.env.GGID_API_KEY,
  });

  const user = await client.getUser(req.params.id);
  res.json(user);
});
```

## Error Handling

```javascript
const { GGIDError, JWTError } = require('@ggid/node');

app.use((err, req, res, next) => {
  if (err instanceof JWTError) {
    return res.status(401).json({ error: 'token_invalid', message: err.message });
  }
  if (err instanceof GGIDError) {
    return res.status(err.statusCode).json({ error: err.code, message: err.message });
  }
  res.status(500).json({ error: 'internal_error' });
});
```

## Environment Variables

```bash
GGID_URL=http://localhost:8080
JWKS_URL=http://localhost:8080/.well-known/jwks.json
GGID_API_KEY=your-api-key
PORT=3000
```

---

*See: [Node SDK Quickstart](../quickstart/node-sdk.md) | [Express Example](../examples/express-integration.md) | [SDK Reference](../sdk-reference.md)*
