# Node.js SDK Quickstart

> Add GGID authentication to any Node.js app in 3 lines.

---

## Install

```bash
npm install @ggid/sdk-node
# or
pnpm add @ggid/sdk-node
```

## Verify a JWT (3 lines)

```javascript
const { GGIDVerifier } = require('@ggid/sdk-node');

const verifier = new GGIDVerifier({ gatewayURL: 'http://localhost:8080', secret: 'your-jwt-secret' });
const claims = await verifier.verify(token);
// claims.tenant_id, claims.sub, claims.scope
```

## Express Middleware

```javascript
const { GGIDMiddleware } = require('@ggid/sdk-node');
const express = require('express');

const app = express();
app.use(GGIDMiddleware({ gatewayURL: 'http://localhost:8080', secret: 'your-jwt-secret' }));

app.get('/api/me', (req, res) => {
    res.json({ user: req.ggid.userID, tenant: req.ggid.tenantID });
});

app.listen(3000);
```

## Full Example

```javascript
const { GGIDClient, GGIDMiddleware } = require('@ggid/sdk-node');
const express = require('express');

const config = { gatewayURL: 'http://localhost:8080', secret: 'your-jwt-secret' };

const app = express();
app.use(GGIDMiddleware(config));

// Protected route
app.get('/api/profile', async (req, res) => {
    const client = new GGIDClient({ ...config, token: req.ggid.token });
    const user = await client.users.get(req.ggid.userID);
    res.json(user);
});

// Role check
app.delete('/api/users/:id', 
    requireScope('delete:users'),  // middleware
    async (req, res) => { /* ... */ }
);

function requireScope(scope) {
    return (req, res, next) => {
        if (!req.ggid.scopes.includes(scope)) {
            return res.status(403).json({ error: 'insufficient_scope' });
        }
        next();
    };
}

app.listen(3000);
```

---

*See: [Express Integration](../integration-guides/express.md) | [SDK Reference](../sdk-reference.md)*