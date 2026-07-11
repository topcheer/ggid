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
npm install express @ggid/node
```

---

## Complete Application

Create `app.js`:

```javascript
const express = require('express');
const { expressAuth, getClaims, requireRole, requirePermission, GGIDClient } = require('@ggid/node');

const app = express();
app.use(express.json());

// ─── Configuration ───────────────────────────────────────────
const GGID_URL = process.env.GGID_URL || 'http://localhost:8080';
const JWKS_URL = process.env.JWKS_URL || `${GGID_URL}/.well-known/jwks.json`;
const TENANT_ID = process.env.TENANT_ID || '00000000-0000-0000-0000-000000000001';
const PORT = process.env.PORT || 3000;

// ─── Auth Middleware ─────────────────────────────────────────
const authConfig = {
  jwksUrl: JWKS_URL,
  issuer: GGID_URL,
};

// ─── Public Routes (no auth) ─────────────────────────────────
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'express-demo' });
});

// ─── Protected API ───────────────────────────────────────────
const api = express.Router();
api.use(expressAuth(authConfig));

// Get current user info from JWT
api.get('/me', (req, res) => {
  const claims = getClaims(req);
  res.json({
    user_id: claims.sub,
    tenant_id: claims.tenant_id,
    email: claims.email,
    scope: claims.scope,
  });
});

// List users — requires read:users permission
api.get('/users', requirePermission({ gatewayUrl: GGID_URL }, 'users', 'read'), async (req, res) => {
  try {
    const claims = getClaims(req);
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      apiKey: process.env.GGID_API_KEY,
    });

    const result = await client.listUsers({ tenant_id: claims.tenant_id });
    res.json({ users: result.users, count: result.users.length });
  } catch (err) {
    console.error('Failed to list users:', err.message);
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Create user — requires admin role
api.post('/users', requireRole('admin'), async (req, res) => {
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
      apiKey: process.env.GGID_API_KEY,
    });

    const user = await client.createUser({ username, email, password });
    res.status(201).json(user);
  } catch (err) {
    if (err.isConflict) {
      return res.status(409).json({ error: 'user_exists' });
    }
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Delete user — requires admin role
api.delete('/users/:id', requireRole('admin'), async (req, res) => {
  try {
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      apiKey: process.env.GGID_API_KEY,
    });

    await client.deleteUser(req.params.id);
    res.json({ status: 'deleted', user_id: req.params.id });
  } catch (err) {
    res.status(502).json({ error: 'upstream_error', message: err.message });
  }
});

// Check permissions via Policy Engine
api.post('/check-permission', async (req, res) => {
  const { action, resource } = req.body;
  const claims = getClaims(req);

  try {
    const client = new GGIDClient({
      gatewayUrl: GGID_URL,
      apiKey: process.env.GGID_API_KEY,
    });

    const result = await client.checkPermission(claims.sub, resource || 'users', action || 'read');
    res.json(result);
  } catch (err) {
    res.status(502).json({ error: 'policy_check_failed', message: err.message });
  }
});

app.use('/api', api);

// ─── Error Handler ───────────────────────────────────────────
app.use((err, req, res, next) => {
  if (err.name === 'JWTError') {
    return res.status(401).json({
      error: 'token_invalid',
      message: err.message,
    });
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
JWKS_URL=http://localhost:8080/.well-known/jwks.json
GGID_API_KEY=your-api-key
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
# → {"user_id":"usr_...","tenant_id":"00000000-...","email":"admin@example.com"}
```

### List Users (requires read:users permission)

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

### Insufficient Role (403)

```bash
# User without admin role
curl -s -X DELETE http://localhost:3000/api/users/usr_123 \
  -H "Authorization: Bearer $JWT" | jq .
# → {"error":"forbidden: requires role 'admin'"}
```

---

## Key Takeaways

1. **`expressAuth(config)`** handles JWT verification on all routes in the group.
2. **`getClaims(req)`** gives you `sub`, `tenant_id`, `email`, `scope` from the JWT.
3. **`requireRole()` / `requirePermission()`** are reusable guards for access control.
4. **`GGIDClient`** lets you call GGID management APIs server-side.
5. **Tenant isolation** is automatic — always use `claims.tenant_id` for database queries.

---

*See also: [Node SDK Quickstart](../quickstart/node-sdk.md) | [Express Integration Guide](../integration-guides/express.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
