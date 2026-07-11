# Node.js SDK Quickstart

> Add GGID authentication to any Node.js app in 3 lines.

---

## Install

```bash
npm install @ggid/node express
# or
yarn add @ggid/node express
```

## Verify a JWT (3 lines)

```javascript
const { JWTVerifier } = require('@ggid/node');

const verifier = new JWTVerifier({ jwksUrl: 'http://localhost:8080/.well-known/jwks.json', issuer: 'http://localhost:8080' });
const claims = await verifier.verify(token);
// claims.sub, claims.tenant_id, claims.email, claims.scope
```

## Express Middleware

```javascript
const { expressAuth, getClaims } = require('@ggid/node');
const express = require('express');

const app = express();
app.use(expressAuth({
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
}));

app.get('/api/me', (req, res) => {
    const claims = getClaims(req);
    res.json({ user: claims.sub, tenant: claims.tenant_id });
});

app.listen(3000);
```

## Full Example

```javascript
const { GGIDClient, expressAuth, requireRole, getClaims } = require('@ggid/node');
const express = require('express');

const config = {
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
};

const app = express();
app.use(expressAuth(config));

// Protected route — get user info
app.get('/api/profile', (req, res) => {
    const claims = getClaims(req);
    res.json({ userId: claims.sub, email: claims.email });
});

// Role-based access control
app.delete('/api/users/:id', requireRole('admin'), async (req, res) => {
    // Only admins can reach here
    res.json({ status: 'deleted' });
});

app.listen(3000);
```

---

*See: [Express Integration](../integration-guides/express.md) | [Express Example](../examples/express-integration.md) | [SDK Reference](../sdk-reference.md)*
