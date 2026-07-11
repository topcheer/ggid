# Express.js Integration Guide

> Add GGID authentication to an Express.js app using the Node SDK.

---

## Install

```bash
npm install @ggid/sdk-node express
```

## Minimal Setup

```javascript
const express = require('express');
const { GGIDMiddleware } = require('@ggid/sdk-node');

const app = express();

// Protect all /api/* routes
app.use('/api', GGIDMiddleware({
  gatewayURL: process.env.GGID_URL || 'http://localhost:8080',
  secret: process.env.JWT_SECRET,
}));

app.get('/api/profile', (req, res) => {
  res.json({
    userID: req.ggid.userID,
    tenantID: req.ggid.tenantID,
    scopes: req.ggid.scopes,
  });
});

app.listen(3000, () => console.log('Server on :3000'));
```

## Scope-Based Authorization

```javascript
function requireScope(scope) {
  return (req, res, next) => {
    if (!req.ggid?.scopes?.includes(scope)) {
      return res.status(403).json({ error: 'insufficient_scope', required: scope });
    }
    next();
  };
}

// Only admins can delete
app.delete('/api/users/:id', requireScope('delete:users'), async (req, res) => {
  // ...
});

// Read access
app.get('/api/users', requireScope('read:users'), async (req, res) => {
  // ...
});
```

## Tenant-Aware Database Queries

```javascript
app.get('/api/items', (req, res) => {
  const tenantID = req.ggid.tenantID;
  db.query('SELECT * FROM items WHERE tenant_id = $1', [tenantID]);
});
```

## Optional Auth (Public + Protected Routes)

```javascript
const { GGIDMiddleware } = require('@ggid/sdk-node');

// Public routes (no middleware)
app.get('/health', (req, res) => res.json({ ok: true }));
app.post('/api/auth/login', loginHandler);

// Protected routes
const authMw = GGIDMiddleware({ gatewayURL: '...', secret: '...' });
app.use('/api', authMw);
```

## Using the GGID Client

```javascript
const { GGIDClient } = require('@ggid/sdk-node');

app.get('/api/users/:id', requireScope('read:users'), async (req, res) => {
  const client = new GGIDClient({
    gatewayURL: process.env.GGID_URL,
    token: req.ggid.token,
  });

  const user = await client.users.get(req.params.id);
  res.json(user);
});
```

## Error Handling

```javascript
app.use((err, req, res, next) => {
  if (err.name === 'GGIDAuthError') {
    return res.status(401).json({ error: 'token_invalid', message: err.message });
  }
  res.status(500).json({ error: 'internal_error' });
});
```

## Environment Variables

```bash
GGID_URL=http://localhost:8080
JWT_SECRET=your-shared-secret
PORT=3000
```

---

*See: [Node SDK Quickstart](../quickstart/node-sdk.md) | [SDK Reference](../sdk-reference.md)*