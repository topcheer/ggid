# @ggid/node

GGID IAM Platform SDK for Node.js — JWT verification, user management, and RBAC.

## Installation

```bash
npm install @ggid/node jose
```

> Requires Node.js 18+ (uses native `fetch`).

## Quick Start

### Express Middleware

Protect routes with JWT verification:

```typescript
import express from 'express';
import { expressAuth, getClaims } from '@ggid/node';

const app = express();

app.use(expressAuth({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
}));

app.get('/profile', (req, res) => {
  const user = getClaims(req); // throws if not authenticated
  res.json({
    id: user.sub,
    email: user.email,
    roles: user.roles,
  });
});

app.listen(3001);
```

The middleware automatically skips authentication for public paths
(`/healthz`, `/api/v1/auth/*`, `/oauth/*`).

### Client API

Server-side management operations:

```typescript
import { GGIDClient } from '@ggid/node';

const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  tenantId: '00000000-0000-0000-0000-000000000001',
  issuer: 'ggid',
});

// Login
const tokens = await client.login('admin', 'Admin@123456');

// List users (requires access token)
const { users } = await client.listUsers(tokens.access_token);

// Check permission
const result = await client.checkPermission(
  tokens.access_token,
  'documents:sensitive',
  'read',
);
console.log(result.allowed); // true/false
```

## Authentication

### Login

```typescript
const tokens = await client.login('username', 'password');
// tokens.access_token — JWT (1h TTL)
// tokens.refresh_token — for token rotation
// tokens.expires_in — TTL in seconds
```

### Register

```typescript
const { user_id } = await client.register(
  'john.doe',
  'john@example.com',
  'SecurePass@123',
  'John Doe', // optional display name
);
```

### Verify JWT

```typescript
// Uses jose library with JWKS caching
const claims = await client.verifyToken(accessToken);
console.log(claims.sub);       // user ID
console.log(claims.email);     // email
console.log(claims.roles);     // role array
console.log(claims.tenant_id); // tenant UUID
```

## User Management

```typescript
// List users
const { users } = await client.listUsers(accessToken, 50);

// Get user by ID
const user = await client.getUser(accessToken, userId);

// Delete user
await client.deleteUser(accessToken, userId);
```

## Role & Permission Management

```typescript
// List roles
const { roles } = await client.listRoles(accessToken);

// Check permission (calls the policy engine)
const result = await client.checkPermission(
  accessToken,
  'documents:sensitive',
  'read',
  userId, // optional, defaults to token's user
);
```

## Middleware

### `expressAuth(config)`

Express middleware that verifies JWT on every request (except public paths).

```typescript
app.use(expressAuth({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
}));
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `jwksUrl` | `string` | Yes | JWKS endpoint URL |
| `issuer` | `string` | No | Expected JWT issuer (verified if set) |

### `requirePermission(resource, action)`

Route-level middleware for permission checks:

```typescript
app.delete('/api/users/:id',
  requirePermission('iam:users', 'delete'),
  async (req, res) => {
    await client.deleteUser(token, req.params.id);
    res.json({ status: 'deleted' });
  },
);
```

### `getClaims(req)`

Extract JWT claims from the request:

```typescript
const claims = getClaims(req);
// claims.sub, claims.email, claims.roles, claims.tenant_id
```

## Types Reference

### `GGIDConfig`

| Field | Type | Required | Default |
|-------|------|----------|---------|
| `gatewayUrl` | `string` | Yes | — |
| `jwksUrl` | `string` | No | — |
| `tenantId` | `string` | No | `00000000-...-001` |
| `issuer` | `string` | No | — |
| `timeout` | `number` | No | `30000` |

### `JWTClaims`

| Field | Type | Description |
|-------|------|-------------|
| `sub` | `string` | User UUID |
| `email` | `string?` | Email |
| `tenant_id` | `string?` | Tenant UUID |
| `roles` | `string[]?` | Role keys |
| `exp` | `number?` | Expiry timestamp |
| `iat` | `number?` | Issued-at timestamp |
| `iss` | `string?` | Issuer |

### `TokenSet`

| Field | Type |
|-------|------|
| `access_token` | `string` |
| `refresh_token` | `string?` |
| `id_token` | `string?` |
| `token_type` | `string` |
| `expires_in` | `number` |

## Framework Integrations

### Fastify

```typescript
import Fastify from 'fastify';

const app = Fastify();

// Use as a preHandler hook
app.addHook('preHandler', async (request, reply) => {
  // Skip public paths
  if (request.url.startsWith('/healthz')) return;

  const auth = request.headers.authorization;
  if (!auth?.startsWith('Bearer ')) {
    return reply.code(401).send({ error: 'missing token' });
  }

  const verifier = new JWTVerifier({
    jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  });
  try {
    const claims = await verifier.verify(auth.slice(7));
    request.ggUser = claims;
  } catch {
    return reply.code(401).send({ error: 'invalid token' });
  }
});
```

### Next.js API Routes

```typescript
// app/api/profile/route.ts
import { JWTVerifier } from '@ggid/node';

const verifier = new JWTVerifier({
  jwksUrl: process.env.GGID_JWKS_URL!,
});

export async function GET(request: Request) {
  const auth = request.headers.get('authorization');
  if (!auth?.startsWith('Bearer ')) {
    return Response.json({ error: 'unauthorized' }, { status: 401 });
  }
  const claims = await verifier.verify(auth.slice(7));
  return Response.json({ user: claims.sub, email: claims.email });
}
```

## License

Apache 2.0
