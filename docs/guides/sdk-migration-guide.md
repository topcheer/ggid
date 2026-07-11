# SDK Migration Guide

> Migrate from Auth0, Keycloak, or Firebase Auth to GGID. Side-by-side API mapping.

---

## From Auth0

### SDK Package

| Auth0 | GGID |
|-------|------|
| `auth0` (npm) | `@ggid/node` (npm) |
| `github.com/auth0/go-jwt-middleware` | `github.com/ggid/ggid/sdk/go` |

### Authentication

**Auth0:**
```javascript
const auth0 = new AuthenticationClient({ domain, clientId, clientSecret });
const token = await auth0.passwordGrant({ username, password });
```

**GGID:**
```javascript
const { GGIDClient } = require('@ggid/node');
const client = new GGIDClient({ gatewayUrl: 'http://localhost:8080', apiKey: '...' });
const result = await client.login(username, password);
// result.access_token, result.refresh_token
```

### JWT Verification

**Auth0:**
```javascript
const { expressjwt: jwt } = require('express-jwt');
app.use(jwt({ secret: jwksClient, audience, issuer }));
```

**GGID:**
```javascript
const { expressAuth, getClaims } = require('@ggid/node');
app.use('/api', expressAuth({
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
}));
const claims = getClaims(req); // claims.sub, claims.tenant_id
```

### User Management

| Auth0 | GGID |
|-------|------|
| `auth0.users.create(data)` | `client.createUser(data)` |
| `auth0.users.get({ id })` | `client.getUser(id)` |
| `auth0.users.update({ id }, data)` | `client.updateUser(id, data)` |
| `auth0.users.delete({ id })` | `client.deleteUser(id)` |
| `auth0.users.getAll()` | `client.listUsers({ tenant_id })` |

### Social Login

| Auth0 | GGID |
|-------|------|
| `connection: 'google-oauth2'` | `provider: 'google'` |
| `connection: 'github'` | `provider: 'github'` |

### Roles & Permissions

| Auth0 | GGID |
|-------|------|
| Management API + Roles endpoint | `POST /api/v1/roles` |
| Assign role: `auth0.roles.assignUsers()` | `POST /api/v1/users/{id}/roles` |
| Check permission in token: `permissions` claim | `POST /api/v1/policies/check` |

---

## From Keycloak

### SDK Package

| Keycloak | GGID |
|----------|------|
| `keycloak-connect` (npm) | `@ggid/node` (npm) |
| Java adapter (`keycloak-core`) | `dev.ggid:ggid-sdk` (Maven) |

### Token Verification

**Keycloak:**
```javascript
const { Keycloak } = require('keycloak-connect');
const kc = new Keycloak({ scope: 'openid' }, keycloakConfig);
app.use(kc.middleware());
app.get('/api', kc.protect(), handler);
```

**GGID:**
```javascript
const { expressAuth, requireRole } = require('@ggid/node');
app.use('/api', expressAuth({ jwksUrl: '...', issuer: '...' }));
app.get('/api', requireRole('admin'), handler);
```

### Realm → Tenant Mapping

| Keycloak | GGID |
|----------|------|
| Realm | Tenant |
| `realm: 'my-realm'` | `X-Tenant-ID: <uuid>` header |
| Realm-specific clients | Tenant-specific API keys |

### Role Mapping

| Keycloak | GGID |
|----------|------|
| Realm role | Role (`POST /api/v1/roles`) |
| Client role | Role with scoped permissions |
| `keycloak.protect('realm-admin')` | `requireRole('admin')` |

---

## From Firebase Auth

### SDK Package

| Firebase | GGID |
|----------|------|
| `firebase-admin` (npm) | `@ggid/node` (npm) |
| `firebase-admin` (PyPI) | `ggid` (PyPI) |

### Token Verification

**Firebase:**
```javascript
const admin = require('firebase-admin');
admin.initializeApp({ credential: admin.credential.cert(serviceAccount) });
const decoded = await admin.auth().verifyIdToken(idToken);
// decoded.uid, decoded.email
```

**GGID:**
```javascript
const { JWTVerifier } = require('@ggid/node');
const verifier = new JWTVerifier({ jwksUrl: '...', issuer: '...' });
const claims = await verifier.verify(idToken);
// claims.sub (uid), claims.email, claims.tenant_id
```

### User Management

| Firebase | GGID |
|----------|------|
| `admin.auth().createUser({ uid, email })` | `client.createUser({ username, email })` |
| `admin.auth().getUser(uid)` | `client.getUser(id)` |
| `admin.auth().updateUser(uid, props)` | `client.updateUser(id, props)` |
| `admin.auth().deleteUser(uid)` | `client.deleteUser(id)` |
| `admin.auth().listUsers()` | `client.listUsers({ tenant_id })` |
| Custom claims: `setCustomUserClaims()` | Roles via `POST /api/v1/users/{id}/roles` |

### Custom Claims

**Firebase:**
```javascript
await admin.auth().setCustomUserClaims(uid, { admin: true, tenant: 'acme' });
```

**GGID:**
```bash
# Assign role (which carries permissions)
curl -X POST http://localhost:8080/api/v1/users/$USER_ID/roles \
  -H "Authorization: Bearer $JWT" \
  -d '{"role_id":"admin-role-id"}'
```

---

## Migration Checklist

- [ ] Deploy GGID infrastructure (PostgreSQL, Redis, NATS)
- [ ] Create tenant for your application
- [ ] Configure JWT signing keys (RSA)
- [ ] Migrate users (bulk import via SCIM or API)
- [ ] Map existing roles/permissions to GGID Policy Engine
- [ ] Update SDK in application code
- [ ] Configure social login providers (Google, GitHub, etc.)
- [ ] Configure SAML/OAuth enterprise SSO if applicable
- [ ] Set up audit logging (NATS JetStream)
- [ ] Test E2E auth flow
- [ ] Remove old SDK dependencies

---

*See: [3-Line Integration](../quickstart/3-line-integration.md) | [SDK Quickstart](../quickstart/sdk-quickstart.md) | [Auth0 Migration Guide](../migration-from-auth0.md)*

*Last updated: 2025-07-11*
