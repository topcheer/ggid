# GGID Node.js SDK Guide

This guide covers installing, configuring, and using the GGID Node.js SDK (`@ggid/node`) for authentication, user management, authorization, and AI agent identity.

## Installation

### npm

```bash
npm install @ggid/node
```

### yarn

```bash
yarn add @ggid/node
```

### pnpm

```bash
pnpm add @ggid/node
```

### From Source

```bash
cd sdk/node
npm install
npm run build
# Output: dist/
```

## Configuration

### Initialize Client

```typescript
import { GGIDClient } from '@ggid/node';

const client = new GGIDClient({
  gatewayUrl: 'https://api.ggid.example.com',
  tenantId: '00000000-0000-0000-0000-000000000001',
  apiKey: process.env.GGID_API_KEY,  // Optional: for admin operations
});
```

### Configuration Options

```typescript
interface GGIDConfig {
  gatewayUrl: string;        // GGID API Gateway URL
  tenantId: string;          // Your tenant UUID
  apiKey?: string;           // API key for admin/service operations
  timeout?: number;          // Request timeout (default: 30000ms)
  retries?: number;          // Retry count (default: 3)
}
```

## Authentication

### Login

```typescript
const tokens = await client.login({
  username: 'user@example.com',
  password: 'SecurePassword123!',
});

// tokens.access_token   → JWT for API calls
// tokens.refresh_token  → Use to refresh after expiry
// tokens.expires_in     → Seconds until expiry
// tokens.token_type     → "Bearer"
```

### Register

```typescript
const { user_id } = await client.register(
  'newuser',
  'newuser@example.com',
  'SecurePassword123!'
);
```

### Refresh Token

```typescript
const newTokens = await client.refreshToken(tokens.refresh_token);
```

### Verify Token

```typescript
const claims = await client.verifyToken(accessToken);
// claims.sub        → User UUID
// claims.tenant_id  → Tenant UUID
// claims.scope      → Space-separated scopes
// claims.exp        → Expiry timestamp
```

### Logout

```typescript
await client.logout(accessToken);
```

## User Management

### Create User

```typescript
const user = await client.createUser({
  username: 'alice',
  email: 'alice@example.com',
  password: 'SecurePass1!',
});
```

### Get User

```typescript
const user = await client.getUser('user-uuid');
```

### Update User

```typescript
const updated = await client.updateUser('user-uuid', {
  email: 'newemail@example.com',
  phone: '+1234567890',
});
```

### Delete User

```typescript
await client.deleteUser('user-uuid');
```

### List Users

```typescript
const page = await client.listUsers({ page: 1, pageSize: 20 });
// page.items       → User[]
// page.total       → Total count
// page.totalPages  → Number of pages
```

## Role Management

### Create Role

```typescript
const role = await client.createRole({
  key: 'developer',
  name: 'Developer',
});
```

> The `key` field must be unique within the tenant. Empty keys cause a 500 error (UNIQUE constraint).

### List Roles

```typescript
const page = await client.listRoles();
for (const role of page.items) {
  console.log(`${role.key}: ${role.name}`);
}
```

### Assign / Remove Role

```typescript
await client.assignRole('user-uuid', 'role-uuid');
await client.removeRole('user-uuid', 'role-uuid');
```

## Organization Management

### Create Organization

```typescript
const org = await client.createOrg({
  name: 'Engineering Team',
  parentId: 'parent-org-uuid',  // Optional for nested orgs
});
```

### List Organizations

```typescript
const page = await client.listOrgs();
```

## Authorization

### Check Permission

```typescript
const result = await client.checkPermission(
  'user-uuid',
  'document:report.pdf',   // resource
  'read'                   // action
);

if (result.allowed) {
  // Grant access
} else {
  // Deny
}
```

## AI Agent Identity

### Register Agent

```typescript
const agent = await client.registerAgent(
  {
    name: 'my-service-agent',
    type: 'service',              // 'service' | 'mcp' | 'workflow'
    scopes: ['users:read', 'users:write'],
    max_delegation_depth: 3,
    mcp_servers: ['https://mcp.internal.com'],
  },
  adminAccessToken  // Requires admin scope
);
```

### List Agents

```typescript
const { agents, total } = await client.listAgents(adminAccessToken);
```

### Exchange Agent Token

```typescript
const agentTokens = await client.exchangeAgentToken(
  {
    agent_id: agent.id,
    scope: 'users:read',
    delegation_chain: ['parent-agent-id'],
  },
  adminAccessToken
);
```

### Verify Agent Token

```typescript
const claims = await client.verifyAgentToken(agentTokens.access_token);
// claims.agent_id           → Agent UUID
// claims.delegation_chain   → string[]
// claims.mcp_servers        → string[]
// claims.max_delegation_depth → number
```

## Express.js Middleware

### JWT Authentication

```typescript
import { authMiddleware } from '@ggid/node';

app.use('/api/protected', authMiddleware({
  gatewayUrl: 'https://api.ggid.example.com',
  tenantId: '00000000-0000-0000-0000-000000000001',
}));

// req.user is now populated with JWT claims
app.get('/api/protected/profile', (req, res) => {
  res.json({ userId: req.user.sub, scopes: req.user.scope });
});
```

### Scope Enforcement

```typescript
import { requireScope } from '@ggid/node';

app.delete('/api/users/:id',
  authMiddleware(config),
  requireScope('users:delete'),
  async (req, res) => {
    await client.deleteUser(req.params.id);
    res.status(204).send();
  }
);
```

## Token Manager (Auto-Refresh)

```typescript
import { TokenManager } from '@ggid/node';

const tokenManager = new TokenManager(client);

// Login and store tokens
await tokenManager.login('user@example.com', 'password');

// Get valid access token (auto-refreshes if expired)
const token = await tokenManager.getAccessToken();

// Use with client calls
const user = await client.getUser('user-uuid');
```

## Error Handling

```typescript
import { GGIDError } from '@ggid/node';

try {
  await client.deleteUser('nonexistent-uuid');
} catch (err) {
  if (err instanceof GGIDError) {
    console.error(`Status: ${err.statusCode}`);
    console.error(`Message: ${err.message}`);

    switch (err.statusCode) {
      case 401: // Unauthorized — invalid/expired token
        break;
      case 403: // Forbidden — insufficient scope
        break;
      case 404: // Not Found
        break;
      case 409: // Conflict — duplicate resource
        break;
      case 429: // Rate limited
        break;
    }
  }
}
```

## Complete Example

```typescript
import { GGIDClient } from '@ggid/node';

async function main() {
  const client = new GGIDClient({
    gatewayUrl: process.env.GGID_GATEWAY_URL!,
    tenantId: process.env.GGID_TENANT_ID!,
  });

  // Login
  const tokens = await client.login('admin@example.com', 'password');

  // Create a user
  const user = await client.createUser({
    username: 'alice',
    email: 'alice@example.com',
    password: 'SecurePass1!',
  });

  // Create and assign role
  const role = await client.createRole({ key: 'dev', name: 'Developer' });
  await client.assignRole(user.id, role.id);

  // Check permission
  const perm = await client.checkPermission(user.id, 'api:read', 'execute');
  console.log('Permission granted:', perm.allowed);

  // Cleanup
  await client.deleteUser(user.id);
  await client.logout(tokens.access_token);
}

main().catch(console.error);
```

## API Reference Summary

| Method                          | Description                    |
|---------------------------------|--------------------------------|
| `login(input)`                  | Authenticate with credentials  |
| `register(username, email, pwd)`| Register new user             |
| `refreshToken(refreshToken)`    | Refresh access token           |
| `verifyToken(token)`            | Verify JWT and extract claims  |
| `logout(token)`                 | Invalidate session             |
| `createUser(input)`             | Create new user                |
| `getUser(id)`                   | Get user by ID                 |
| `updateUser(id, input)`         | Update user attributes         |
| `deleteUser(id)`                | Delete user                    |
| `listUsers(opts?)`              | List users (paginated)         |
| `createRole(input)`             | Create role                    |
| `listRoles(opts?)`              | List roles                     |
| `assignRole(userId, roleId)`    | Assign role to user            |
| `removeRole(userId, roleId)`    | Remove role from user          |
| `createOrg(input)`              | Create organization            |
| `listOrgs(opts?)`               | List organizations             |
| `checkPermission(uid, res, act)`| Check authorization            |
| `registerAgent(input, token)`   | Register AI agent              |
| `listAgents(token)`             | List registered agents         |
| `exchangeAgentToken(input, tok)`| Exchange agent token           |
| `verifyAgentToken(token)`       | Verify agent JWT               |

## See Also

- [Java SDK Guide](java-sdk-guide.md)
- [Quick Start](quick-start.md)
- [API Reference](api-reference.md)
- [AI Agent Identity](ai-agent-identity.md)
