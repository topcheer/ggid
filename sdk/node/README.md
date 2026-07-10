# @ggid/node

GGID IAM Platform SDK for Node.js — JWT verification, user management, and RBAC.

## Installation

```bash
npm install @ggid/node jose
```

## Quick Start

### Express

```typescript
import express from 'express';
import { expressAuth, getClaims } from '@ggid/node';

const app = express();

app.use(expressAuth({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
}));

app.get('/profile', (req, res) => {
  const user = getClaims(req);
  res.json({ user });
});
```

### Client API

```typescript
import { GGIDClient } from '@ggid/node';

const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  tenantId: '00000000-0000-0000-0000-000000000001',
});

// Login
const tokens = await client.login('admin', 'Admin@123456');

// List users
const { users } = await client.listUsers(tokens.access_token);

// Check permission
const result = await client.checkPermission(
  tokens.access_token,
  'documents:sensitive',
  'read',
);
```

## License

Apache 2.0
