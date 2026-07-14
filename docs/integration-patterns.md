# Integration Patterns

Common integration patterns for connecting GGID with external systems:
webhook consumer, SCIM provisioning, SAML federation, OIDC relying party,
LDAP sync, event-driven audit, and SDK usage in microservices.

---

## Table of Contents

- [Webhook Consumer Pattern](#webhook-consumer-pattern)
- [SCIM Provisioning Pattern](#scim-provisioning-pattern)
- [SAML Federation Pattern](#saml-federation-pattern)
- [OIDC Relying Party Pattern](#oidc-relying-party-pattern)
- [LDAP Directory Sync Pattern](#ldap-directory-sync-pattern)
- [Event-Driven Audit Pattern](#event-driven-audit-pattern)
- [SDK in Microservices Pattern](#sdk-in-microservices-pattern)

---

## Webhook Consumer Pattern

Your application receives GGID events via HTTP webhook.

```
GGID ──POST──► Your Webhook Endpoint
  │                    │
  │ HMAC-SHA256        │ 1. Verify signature
  │ signature          │ 2. Parse event
  │                    │ 3. Idempotency check
  │                    │ 4. Process event
  │                    │ 5. Return 200
  │◄─── 200 OK ────────┤
```

### Implementation (Node.js)

```javascript
const crypto = require('crypto');
const express = require('express');

const app = express();

// Use raw body for signature verification
app.post('/webhooks/ggid', express.raw({type: 'application/json'}), (req, res) => {
  // 1. Verify HMAC signature
  const sig = req.headers['x-ggid-signature'];
  const expected = crypto
    .createHmac('sha256', process.env.GGID_WEBHOOK_SECRET)
    .update(req.body)
    .digest('hex');

  if (`sha256=${expected}` !== sig) {
    return res.status(401).send('Invalid signature');
  }

  // 2. Parse event
  const event = JSON.parse(req.body);

  // 3. Idempotency check
  4. // Process event
  switch (event.event_type) {
    case 'user.created':
      await provisionUserInLocalDB(event.data);
      break;
    case 'user.deleted':
      await deactivateUserInLocalDB(event.data);
      break;
    case 'role.assigned':
      await updateLocalPermissions(event.data);
      break;
  }

  // 5. Acknowledge
  res.status(200).send('OK');
});
```

### When to Use

- Sync user data to your application database
- Trigger workflows on user lifecycle events
- Send notifications via your own notification system
- Maintain a denormalized user cache

---

## SCIM Provisioning Pattern

Okta/Azure AD provisions users into GGID via SCIM 2.0.

```
IdP (Okta)                    GGID (SCIM Endpoint)
  │                                 │
  │ POST /scim/v2/Users             │
  ├────────────────────────────────►│
  │ 201 Created                     │
  │◄────────────────────────────────┤
  │                                 │
  │ PATCH /scim/v2/Users/{id}       │
  │ (add to group, update attrs)    │
  ├────────────────────────────────►│
  │ 200 OK                          │
  │◄────────────────────────────────┤
```

### When to Use

- Enterprise customer uses Okta/Azure AD as their IdP
- Automated user provisioning/deprovisioning
- Keep user attributes in sync

---

## SAML Federation Pattern

GGID trusts an external SAML IdP for authentication.

```
User          Your App (SP)         External IdP
  │ 1. Access    │                       │
  ├────────────►│                       │
  │              │ 2. SAML AuthnRequest  │
  │              ├──────────────────────►│
  │ 3. Login     │                       │
  ├─────────────────────────────────────►│
  │              │ 4. SAML Response      │
  │              │ (signed assertion)    │
  │              │◄──────────────────────┤
  │              │ 5. Verify signature   │
  │              │    Create session     │
  │ 6. App loaded│                       │
  │◄────────────┤                       │
```

### When to Use

- Enterprise SSO (large organizations)
- Customers with existing SAML IdP (AD FS, Shibboleth)
- Legal/compliance requirement for enterprise SSO

---

## OIDC Relying Party Pattern

Your application delegates authentication to GGID via OIDC.

```
User         Your App             GGID (OIDC Provider)
  │ 1. Access   │                       │
  ├────────────►│                       │
  │              │ 2. Redirect to GGID   │
  │              ├──────────────────────►│
  │ 3. Login     │                       │
  ├─────────────────────────────────────►│
  │              │ 4. Auth code + PKCE   │
  │              │◄──────────────────────┤
  │              │ 5. Exchange code      │
  │              ├──────────────────────►│
  │              │ 6. ID Token + AT      │
  │              │◄──────────────────────┤
  │              │ 7. Verify ID Token    │
  │ 8. Logged in │                       │
  │◄────────────┤                       │
```

### When to Use

- Your application needs federated authentication
- Multiple apps sharing one identity provider
- Social login aggregation

---

## LDAP Directory Sync Pattern

GGID reads user/group data from Active Directory or OpenLDAP.

```
AD / OpenLDAP               GGID
  │                            │
  │ 1. Scheduled sync (hourly) │
  │◄───────────────────────────┤
  │ 2. Users + Groups          │
  ├───────────────────────────►│
  │                            │ 3. Map attributes
  │                            │ 4. Create/update users
  │                            │ 5. Sync group memberships
  │                            │ 6. Assign roles via group mapping
```

### When to Use

- Organization uses Active Directory as source of truth
- Need real-time auth + periodic sync
- Group-based role assignment

---

## Event-Driven Audit Pattern

GGID publishes audit events to NATS, external systems subscribe.

```
GGID Services                NATS JetStream              SIEM / Splunk
  │                              │                          │
  │ audit.events.{tenant}        │                          │
  ├─────────────────────────────►│                          │
  │                              │ 1. Consumer subscribes   │
  │                              ├─────────────────────────►│
  │                              │ 2. Events delivered      │
  │                              │◄─────────────────────────┤
  │                              │ 3. ACK                   │
```

### When to Use

- Forward audit events to SIEM (Splunk, Datadog, Elastic)
- Real-time security monitoring
- Compliance log aggregation

---

## SDK in Microservices Pattern

Microservices use the GGID SDK to verify JWTs and call GGID APIs.

```
┌───────────────────────────────────────────────────────┐
│                    API Gateway                         │
│  (JWT verification via GGID JWKS)                      │
└───────┬───────────┬───────────┬───────────┬───────────┘
        │           │           │           │
   ┌────▼───┐ ┌────▼───┐ ┌────▼───┐ ┌────▼───┐
   │Service A│ │Service B│ │Service C│ │Service D│
   │(Go SDK) │ │(Node SDK│ │(Go SDK) │ │(Java SDK│
   └─────────┘ └────────┘ └─────────┘ └────────┘
        │                                              │
        └─────────── GGID SDK ─────────────────────────┘
                    1. Verify JWT (offline, JWKS cached)
                    2. Extract tenant_id, user_id, scopes
                    3. Call GGID Admin API (when needed)
                    4. Evaluate policies (via Policy API)
```

### JWT Verification (Offline)

Microservices verify JWTs locally without calling GGID on every request:

1. Fetch GGID JWKS (cached, refreshed every 5 min)
2. Verify JWT signature using public key
3. Check `exp`, `iss`, `aud`
4. Extract `sub`, `tenant_id`, `scope`
5. Check revocation list (if introspection configured)

### When to Use

- Microservice architecture with multiple services
- Each service needs to verify tokens independently
- Services need to call GGID Admin API for user/role data

---

## Pattern Selection Matrix

| Need | Pattern |
|------|---------|
| Push events to your app | Webhook Consumer |
| IdP → GGID user sync | SCIM Provisioning |
| Enterprise SSO | SAML Federation |
| Your app → GGID auth | OIDC Relying Party |
| AD/OpenLDAP integration | LDAP Directory Sync |
| Forward audit to SIEM | Event-Driven Audit |
| Microservice auth | SDK Pattern |
