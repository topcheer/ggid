# Express.js Integration Example

> Complete, runnable Express.js application with GGID JWT verification, role-based access control, and user info retrieval.

---

## Prerequisites

- Node.js 18+
- GGID Gateway running at `http://localhost:8080`

---

## Project Setup

```bash
mkdir ggid-express-demo && cd ggid-express-demo
npm init -y
npm install express @ggid/node jsonwebtoken
```

---

## Complete Application

Create `app.js`:

```javascript
const express = require('express');
const { expressAuth, GGIDClient } = require('@ggid/node');

const app = express();
app.use(express.json());

// ─── Configuration ───────────────────────────────────────────
const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const JWT_SECRET = process.env.JWT_SECRET || 'your-shared-secret';
const TENANT_ID = process.env.TENANT_ID || '00000000-0000-0000-0000-000000000001';
const PORT = process.env.PORT || 3000;

// ─── Auth Middleware ─────────────────────────────────────────
const authMw = expressAuth({
  jwksUrl: `${GGID_URL}/.well-known/jwks.json`,
  issuer: GGID_URL,
});

// ─── Scope Guard ─────────────────────────────────────────────
function requireScope(scope) {
  return (req, res, next) => {
    if (!req.ggidUser?.scopes?.includes(scope)) {
      return res.status(403).json({
        error: 'insufficient_scope',
        required: scope,
        user_scopes: req.ggidUser?.scopes || [],
      });
    }
    next();
  };
}

// ─── Public Routes (no auth) ─────────────────────────────────
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'express-demo' });
});

// ─── Protected API ───────────────────────────────────────────
const api = express.Router();
api.use(authMw);

// Get current user info from JWT
api.get('/me', (req, res) => {
  res.json({
    user_id: req.ggidUser?.sub,
    tenant_id: req.ggidUser?.tenant_id,
    scopes: req.ggidUser?.scopes,
  });
});

// List users — requires read:users scope
api.get('/users', requireScope('read:users'), async (req, res) => {
  try {
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      tenantId: req.ggidUser?.tenant_id || TENANT_ID,
    });

    // Tenant-scoped query
    const result = await client.listUsers();
    const users = result.items || result;
  } catch (err) {
    console.error('Failed to list users:', err.message);
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Create user — requires write:users scope
api.post('/users', requireScope('write:users'), async (req, res) => {
  const { username, email, password } = req.body;

  if (!username || !email || !password) {
    return res.status(400).json({
      error: 'validation_error',
      message: 'username, email, and password are required',
    });
  }

  try {
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      tenantId: req.ggidUser?.tenant_id || TENANT_ID,
    });

    const user = await client.createUser({
      username,
      email,
      password,
    });

    res.status(201).json(user);
  } catch (err) {
    if (err.status === 409) {
      return res.status(409).json({ error: 'user_exists' });
    }
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Delete user — requires delete:users scope (admin only)
api.delete('/users/:id', requireScope('delete:users'), async (req, res) => {
  try {
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      tenantId: req.ggidUser?.tenant_id || TENANT_ID,
    });

    await client.deleteUser(req.params.id);
    res.json({ status: 'deleted', user_id: req.params.id });
  } catch (err) {
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Check permissions via Policy Engine
api.post('/check-permission', async (req, res) => {
  const { action, resource, resource_type } = req.body;

  try {
    const resp = await fetch(`${GGID_URL}/api/v1/policies/check`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${req.headers.authorization?.replace('Bearer ', '')}`,
        'X-Tenant-ID': req.ggid.tenantID,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        user_id: req.ggid.userID,
        action: action || 'read',
        resource: resource || 'users',
        resource_type: resource_type || 'users',
      }),
    });

    const result = await resp.json();
    res.json(result);
  } catch (err) {
    res.status(502).json({ error: 'policy_check_failed', message: err.message });
  }
});

app.use('/api', api);

// ─── Error Handler ───────────────────────────────────────────
app.use((err, req, res, next) => {
  if (err.name === 'GGIDAuthError') {
    return res.status(401).json({
      error: 'token_invalid',
      message: err.message,
    });
  }
  if (err.name === 'UnauthorizedError') {
    return res.status(401).json({ error: 'unauthorized', message: err.message });
  }
  console.error('Unhandled error:', err);
  res.status(500).json({ error: 'internal_error' });
});

// ─── Start ───────────────────────────────────────────────────
app.listen(PORT, () => {
  console.log(`Express demo running on http://localhost:${PORT}`);
  console.log(`GGID Gateway: ${GGID_URL}`);
  console.log(`Tenant: ${TENANT_ID}`);
});
```

---

## Environment Variables

Create `.env`:

```bash
GGID_URL=http://localhost:8080
JWT_SECRET=your-shared-secret
TENANT_ID=00000000-0000-0000-0000-000000000001
PORT=3000
```

---

## Run

```bash
node app.js
# → Express demo running on http://localhost:3000
```

---

## Test the Endpoints

### Health Check (public)

```bash
curl http://localhost:3000/health
# → {"status":"ok","service":"express-demo"}
```

### Protected Route Without Token (401)

```bash
curl http://localhost:3000/api/me
# → {"error":"missing or invalid token"}
```

### Get User Info

```bash
# First, login to get a JWT
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r .access_token)

# Call protected endpoint
curl -s http://localhost:3000/api/me \
  -H "Authorization: Bearer $JWT" | jq .
# → {"user_id":"usr_...","tenant_id":"00000000-...","scopes":["read:users","write:users"]}
```

### List Users (requires read:users scope)

```bash
curl -s http://localhost:3000/api/users \
  -H "Authorization: Bearer $JWT" | jq .
```

### Check Permission

```bash
curl -s -X POST http://localhost:3000/api/check-permission \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"action":"write","resource":"users"}' | jq .
# → {"allowed":true,"reason":"role_permission_match"}
```

### Insufficient Scope (403)

```bash
# User without delete:users scope
curl -s -X DELETE http://localhost:3000/api/users/usr_123 \
  -H "Authorization: Bearer $JWT" | jq .
# → {"error":"insufficient_scope","required":"delete:users","user_scopes":["read:users"]}
```

---

## Key Takeaways

1. **`GGIDMiddleware`** handles JWT verification on all routes in the group.
2. **`req.ggid`** gives you `userID`, `tenantID`, `scopes`, and `token`.
3. **`requireScope()`** is a reusable guard for fine-grained access control.
4. **`GGIDClient`** lets you call GGID APIs on behalf of the authenticated user.
5. **Tenant isolation** is automatic — always use `req.ggid.tenantID` for database queries.

---

*See also: [Node SDK Quickstart](../quickstart/node-sdk.md) | [Express Integration Guide](../integration-guides/express.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
