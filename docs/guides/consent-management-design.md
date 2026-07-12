# Consent Management Design

Granular consent, per-scope consent, expiry, revocation, consent receipts, dynamic consent UI, audit trail, and GDPR/CCPA compliance.

## Overview

Consent management records what data a user has agreed to share with which application, for what purpose, and allows revocation at any time.

## Consent Model

### Data Flow

```
OAuth Client requests scopes
    │
    ▼
GGID evaluates consent state
    ├── No prior consent → Show consent screen
    ├── Prior consent covers requested scopes → Skip (unless re-consent required)
    └── Prior consent insufficient → Show consent for new scopes only
    │
    ▼
User grants/denies
    │
    ▼
Consent recorded + receipt issued
```

## Consent Record

```json
{
  "consent_id": "cons-uuid",
  "user_id": "uuid",
  "client_id": "oauth-client-123",
  "scopes": ["openid", "profile", "email", "users:read"],
  "purpose": "Profile access for dashboard app",
  "granted_at": "2025-01-15T10:00:00Z",
  "expires_at": "2025-07-15T10:00:00Z",
  "revocable": true,
  "consent_receipt_uri": "https://auth.ggid.dev/consent/cons-uuid/receipt"
}
```

## Per-Scope Consent

Each scope is consented individually — user can grant some and deny others:

```bash
# Client requests 4 scopes, user grants 3
POST /api/v1/auth/consent
{
  "consent_challenge": "challenge-token",
  "granted_scopes": ["openid", "profile", "email"],
  "denied_scopes": ["users:read"],
  "decision": "partial"
}
# → Authorization proceeds with granted scopes only
```

## Consent Expiry

| Scope Type | Default Expiry | Re-consent |
|-----------|---------------|------------|
| Low risk (openid, profile) | 1 year | After expiry |
| Medium risk (users:read) | 6 months | After expiry |
| High risk (users:write) | 3 months | Required every time |
| Financial (payments) | Single session | Every authorization |

```go
func needsReconsent(consent Consent, requestedScopes []string) bool {
    if time.Now().After(consent.ExpiresAt) { return true }
    
    for _, scope := range requestedScopes {
        if !contains(consent.Scopes, scope) { return true }
        if isHighRisk(scope) { return true } // Always re-consent
    }
    return false
}
```

## Consent Revocation

```bash
# User revokes consent entirely
DELETE /api/v1/consent/{consent_id}
# → All tokens from this consent revoked immediately

# User revokes specific scope
DELETE /api/v1/consent/{consent_id}/scopes/users:read
# → Tokens re-issued with reduced scope on next refresh
```

### Revocation Effects

| Action | Effect |
|--------|--------|
| Revoke consent | All access + refresh tokens from that consent invalidated |
| Revoke scope | Future tokens exclude that scope; current tokens valid until expiry |
| User deletes account | All consents revoked automatically |

## Consent Receipts

Every consent grant generates a receipt:

```json
{
  "receipt_id": "rcpt-uuid",
  "jurisdiction": "EU",
  "consent_timestamp": "2025-01-15T10:00:00Z",
  "collection_method": "oauth_consent_screen",
  "data_controller": "Acme Corp",
  "purposes": [
    {"purpose": "identity", "data": ["email", "name"], "lawful_basis": "consent"}
  ],
  "consent_receipt_uri": "https://auth.ggid.dev/consent/receipts/rcpt-uuid"
}
```

Receipts are immutable and stored for 7 years (compliance evidence).

```bash
# List user's consent receipts
GET /api/v1/consent/receipts?user_id=uuid
```

## Dynamic Consent UI

### Consent Screen Generation

```tsx
function ConsentScreen({ client, requestedScopes, existingScopes }) {
  const newScopes = requestedScopes.filter(s => !existingScopes.includes(s));
  
  return (
    <Dialog>
      <Header>
        <img src={client.logoUri} />
        <h2>{client.clientName} wants to access:</h2>
      </Header>
      <ScopeList>
        {newScopes.map(scope => (
          <ScopeRow
            key={scope}
            name={scope}
            description={SCOPE_DESCRIPTIONS[scope]}
            icon={getScopeIcon(scope)}
            risk={getScopeRisk(scope)}
          />
        ))}
      </ScopeList>
      <Footer>
        <Button variant="danger" onClick={deny}>Deny</Button>
        <Button variant="primary" onClick={allowAll}>Allow All</Button>
      </Footer>
    </Dialog>
  );
}
```

### Scope Descriptions

```yaml
scope_descriptions:
  openid: "Verify your identity"
  profile: "Read your name and profile"
  email: "Read your email address"
  users:read: "View users in your organization"
  users:write: "Create and modify users"
  users:delete: "Delete users (high risk)"
  roles:assign: "Assign roles to users (high risk)"
```

Descriptions are shown in plain language, not technical scope names.

## Granular Consent (Data Minimization)

```bash
# Client requests broad scope, user narrows
POST /api/v1/auth/consent
{
  "consent_challenge": "...",
  "granted_scopes": ["openid"],
  "attribute_selection": {
    "profile": ["display_name"]  // Only name, not locale/timezone
  }
}
```

User selects which specific attributes to release, even within an approved scope.

## Audit Trail

```json
{
  "event": "consent.granted",
  "user_id": "uuid",
  "client_id": "oauth-client-123",
  "scopes": ["openid", "profile"],
  "denied_scopes": ["email"],
  "timestamp": "2025-01-15T10:00:00Z",
  "ip": "10.0.1.5",
  "user_agent": "Mozilla/5.0..."
}
```

Events: `consent.granted`, `consent.denied`, `consent.revoked`, `consent.expired`, `consent.scope_added`.

## GDPR / CCPA Compliance

| Requirement | GGID Implementation |
|-------------|---------------------|
| Right to withdraw consent | `DELETE /api/v1/consent/{id}` |
| Right to access consent records | `GET /api/v1/consent?user_id=...` |
| Purpose limitation | Each consent tied to stated purpose |
| Data minimization | Per-attribute selection |
| Audit trail | Immutable consent receipts |
| Easy withdrawal | One-click revoke in user dashboard |
| Pre-ticked boxes prohibited | All scopes unchecked by default |

### Global Privacy Control (GPC)

```go
// Respect GPC signal as opt-out
if r.Header.Get("Sec-GPC") == "1" {
    // Treat as consent withdrawal for data sharing
    revokeDataSharingConsent(userID)
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Consent denial rate | >20% → client requesting too many scopes |
| Consent expiry spikes | Batch expiry → notify users |
| Revocation rate | >10% → UX or trust issue |
| Partial consent rate | High → clients over-requesting |

## See Also

- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
- [Privacy by Design](privacy-by-design.md)
- [GDPR Compliance](gdpr-compliance.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
