# GGID Node.js SDK

A production-ready Node.js/TypeScript client SDK for the [GGID](https://github.com/ggid/ggid) IAM platform.

## Installation

```bash
npm install @ggid/node
# or
yarn add @ggid/node
```

### Peer Dependencies

The Express middleware requires `express`:

```bash
npm install express
```

## Quick Start

```typescript
import { GGIDClient } from '@ggid/node';

// Create a client.
const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  apiKey: 'your-api-key',
});

// Login.
const tokens = await client.login({
  username: 'alice',
  password: 'SecurePass@123',
});
console.log('Access token:', tokens.access_token);

// Create a user.
const user = await client.createUser({
  username: 'bob',
  email: 'bob@example.com',
  password: 'SecurePass@123',
});
console.log('Created user:', user.id);

// Check permission.
const result = await client.checkPermission(user.id, 'documents', 'read');
console.log('Allowed:', result.allowed);
```

## Authentication

### JWT Verification

```typescript
import { GGIDClient } from '@ggid/node';

const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'https://iam.example.com',
});

// Verify a JWT (signature checked via JWKS).
const claims = await client.verifyToken(accessToken);
console.log('User:', claims.sub, claims.email);
```

### Token Refresh

```typescript
const tokens = await client.refreshToken(refreshToken);
// tokens.access_token + tokens.refresh_token (rotated)
```

## User Management

```typescript
// Create
const user = await client.createUser({
  username: 'alice',
  email: 'alice@example.com',
  password: 'SecurePass@123',
});

// Get
const user = await client.getUser('user-id');

// Update
const updated = await client.updateUser('user-id', {
  email: 'new@example.com',
});

// Delete
await client.deleteUser('user-id');

// List with pagination
const result = await client.listUsers({
  page: 1,
  page_size: 20,
  search: 'alice',
});

// Role assignment
await client.assignRole('user-id', 'role-id');
await client.removeRole('user-id', 'role-id');
```

## Express Middleware

### JWT Authentication

```typescript
import express from 'express';
import { expressAuth } from '@ggid/node';

const app = express();

// Protect all routes with JWT verification.
app.use(expressAuth({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'https://iam.example.com',
}));
```

### Role-Based Access Control

```typescript
import { requireRole } from '@ggid/node';

// Require 'admin' role (checked from JWT claims, no API call).
app.delete('/api/users/:id', requireRole('admin'), deleteUserHandler);
```

### Permission-Based Access Control

```typescript
import { requirePermission } from '@ggid/node';

// Check permission via the GGID policy engine.
app.get('/api/documents', requirePermission(
  { gatewayUrl: 'https://iam.example.com' },
  'documents',
  'read',
), listDocumentsHandler);
```

### Access User Info in Handlers

```typescript
import { getClaims } from '@ggid/node';

app.get('/api/me', (req, res) => {
  const claims = getClaims(req);
  res.json({ userId: claims.sub, email: claims.email });
});
```

## Error Handling

```typescript
import { GGIDError } from '@ggid/node';

try {
  await client.getUser('nonexistent');
} catch (err) {
  if (err instanceof GGIDError) {
    if (err.isNotFound) {
      // 404
    } else if (err.isRateLimited) {
      // 429
    } else if (err.isConflict) {
      // 409
    }
    console.log(err.statusCode, err.code, err.message);
  }
}
```

## API Reference

| Method | Description |
|--------|-------------|
| `login(input)` | Authenticate with username/password |
| `register(username, email, password)` | Register a new user |
| `logout(accessToken)` | Invalidate an access token |
| `refreshToken(refreshToken)` | Refresh an access token |
| `verifyToken(token)` | Verify JWT and return claims |
| `createUser(input)` | Create a new user |
| `getUser(userId)` | Get user by ID |
| `updateUser(userId, input)` | Update user fields |
| `deleteUser(userId)` | Delete a user |
| `listUsers(opts?)` | List users with pagination |
| `assignRole(userId, roleId)` | Assign role to user |
| `removeRole(userId, roleId)` | Remove role from user |
| `createRole(input)` | Create a role |
| `listRoles(opts?)` | List roles |
| `createOrg(input)` | Create an organization |
| `listOrgs(opts?)` | List organizations |
| `checkPermission(userId, resource, action)` | Check authorization |

## License

Apache 2.0
