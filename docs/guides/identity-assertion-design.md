# Identity Assertion Design

This guide covers SAML assertion structure, OIDC ID token structure, claim-based assertions, attribute statements, assertion lifetime and freshness, audience restriction, conditions, and GGID's assertion implementation.

## SAML Assertion Structure

### Full Assertion XML

```xml
<saml:Assertion ID="_abc123" IssueInstant="2026-07-12T10:00:00Z" Version="2.0">
  <saml:Issuer>https://auth.ggid.example.com</saml:Issuer>
  <ds:Signature>...</ds:Signature>
  <saml:Subject>
    <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
      user@example.com
    </saml:NameID>
    <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml:SubjectConfirmationData NotOnOrAfter="2026-07-12T10:05:00Z"
        Recipient="https://sp.example.com/saml/acs" InResponseTo="_def456"/>
    </saml:SubjectConfirmation>
  </saml:Subject>
  <saml:Conditions NotBefore="2026-07-12T09:59:00Z" NotOnOrAfter="2026-07-12T10:10:00Z">
    <saml:AudienceRestriction>
      <saml:Audience>https://sp.example.com</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>
  <saml:AuthnStatement AuthnInstant="2026-07-12T10:00:00Z" SessionIndex="_session789">
    <saml:AuthnContext>
      <saml:AuthnContextClassRef>
        urn:oasis:names:tc:SAML:2.0:ac:classes:TimeSyncToken
      </saml:AuthnContextClassRef>
    </saml:AuthnContext>
  </saml:AuthnStatement>
  <saml:AttributeStatement>
    <saml:Attribute Name="email">
      <saml:AttributeValue>user@example.com</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="role">
      <saml:AttributeValue>admin</saml:AttributeValue>
    </saml:Attribute>
  </saml:AttributeStatement>
</saml:Assertion>
```

### Element Breakdown

| Element | Purpose | Required |
|---|---|---|
| `Issuer` | Identifies the IdP | Yes |
| `Signature` | XML digital signature | Yes |
| `Subject` | Who the assertion is about | Yes |
| `NameID` | Subject identifier | Yes |
| `SubjectConfirmation` | How subject was authenticated | Yes |
| `Conditions` | Validity constraints | Yes |
| `AudienceRestriction` | Who can consume the assertion | Recommended |
| `AuthnStatement` | When and how authentication happened | Yes |
| `AttributeStatement` | Additional user attributes | Optional |

## OIDC ID Token

### JWT Structure

```
header.payload.signature
```

### Header

```json
{"alg": "RS256", "typ": "JWT", "kid": "2026-07"}
```

### Payload (Claims)

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid-1234",
  "aud": "client-id-5678",
  "exp": 1700000600,
  "iat": 1700000000,
  "nonce": "client-nonce-abc",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "roles": ["admin", "user-admin"],
  "tenant_id": "tenant-uuid-9012",
  "acr": "urn:oasis:names:tc:SAML:2.0:ac:classes:TimeSyncToken",
  "amr": ["pwd", "otp"]
}
```

### Standard Claims

| Claim | Description | Source |
|---|---|---|
| `iss` | Issuer URL | Configured |
| `sub` | Subject (user ID) | User database |
| `aud` | Audience (client ID) | Request |
| `exp` | Expiration time | Configured (15min) |
| `iat` | Issued at time | Current time |
| `nonce` | Client nonce | Client request |
| `email` | User email | User database |
| `acr` | Auth context class ref | Auth method |
| `amr` | Auth method references | Auth methods used |

## SAML vs OIDC Comparison

| Concept | SAML | OIDC |
|---|---|---|
| Subject identifier | NameID | sub claim |
| User attributes | AttributeStatement | ID Token claims |
| Auth context | AuthnContextClassRef | acr claim |
| Auth methods | (not standardized) | amr claim |
| Issuer | Issuer element | iss claim |
| Audience | AudienceRestriction | aud claim |
| Expiry | NotOnOrAfter | exp claim |
| Issued time | IssueInstant | iat claim |
| Verification | XML signature | JWT signature |

## Assertion Lifetime and Freshness

### SAML Timing

| Element | Purpose | Typical Value |
|---|---|---|
| `IssueInstant` | When assertion was issued | Current time |
| `NotBefore` | Earliest valid time | IssueInstant - 60s (skew) |
| `NotOnOrAfter` | Expiry time | IssueInstant + 5-10min |

### OIDC Timing

| Claim | Purpose | Typical Value |
|---|---|---|
| `iat` | Issued at | Current time |
| `nbf` | Not before | iat (or iat - 60s) |
| `exp` | Expiration | iat + 15min |

### Replay Prevention

```go
func preventReplay(assertionID string) error {
    key := "assertion:used:" + assertionID
    set, err := redis.SetNX(ctx, key, "1", 10*time.Minute)
    if err != nil { return err }
    if !set { return ErrAssertionReplay }
    return nil
}
```

## Audience Restriction

```go
func validateAudience(audiences []string, expected string) error {
    for _, aud := range audiences {
        if aud == expected { return nil }
    }
    return ErrAudienceNotMatched
}
```

## Conditions

### NotBefore / NotOnOrAfter

```go
func validateConditions(notBefore, notOnOrAfter time.Time, skew time.Duration) error {
    if time.Now().Add(-skew).Before(notBefore) {
        return ErrNotYetValid
    }
    if time.Now().Add(skew).After(notOnOrAfter) {
        return ErrExpired
    }
    return nil
}
```

## GGID Assertion Implementation

### SAML Assertion Generation

```go
func (s *SAMLService) GenerateAssertion(user *User, sp *SAMLSP) (*saml.Assertion, error) {
    now := time.Now()
    assertion := &saml.Assertion{
        ID:           "_" + uuid.New().String(),
        IssueInstant: now,
        Version:      "2.0",
        Issuer:       &saml.Issuer{Value: s.config.EntityID},
        Subject: &saml.Subject{
            NameID: &saml.NameID{Format: sp.NameIDFormat, Value: user.Email},
        },
        Conditions: &saml.Conditions{
            NotBefore:    now.Add(-60 * time.Second),
            NotOnOrAfter: now.Add(5 * time.Minute),
            AudienceRestriction: &saml.AudienceRestriction{Audience: sp.EntityID},
        },
        AuthnStatement: &saml.AuthnStatement{
            AuthnInstant: now,
            AuthnContext: &saml.AuthnContext{
                AuthnContextClassRef: s.getACRForUser(user),
            },
        },
        AttributeStatement: s.buildAttributeStatement(user, sp),
    }
    return s.signAssertion(assertion)
}
```

### OIDC ID Token Generation

```go
func (s *OIDCService) GenerateIDToken(user *User, client *Client, nonce string) (string, error) {
    now := time.Now()
    claims := jwt.MapClaims{
        "iss": s.config.Issuer, "sub": user.ID, "aud": client.ID,
        "exp": now.Add(15 * time.Minute).Unix(), "iat": now.Unix(),
        "nonce": nonce, "email": user.Email, "email_verified": user.EmailVerified,
        "name": user.Name, "roles": user.Roles, "tenant_id": user.TenantID,
        "acr": s.getACRForUser(user), "amr": s.getAMRForUser(user),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.config.KeyID
    return token.SignedString(s.config.PrivateKey)
}
```

### Configuration

```yaml
assertion:
  saml:
    lifetime: 5m
    clock_skew: 60s
    require_signed: true
    replay_prevention: true
  oidc:
    id_token_lifetime: 15m
    signing_algorithm: "RS256"
```

## Best Practices

1. **Short assertion lifetime** — 5-10 min SAML, 15 min OIDC
2. **Always sign assertions** — Never send unsigned
3. **Enforce audience restriction** — Prevent wrong-audience consumption
4. **Prevent replay** — Track used assertion IDs
5. **Allow clock skew** — 60 seconds standard
6. **Include auth context** — ACR and AMR for step-up
7. **Minimize attributes** — Only include needed attributes
8. **Verify before use** — Validate signature, expiry, audience
9. **Encrypt for high-security** — Encrypt SAML assertions
10. **Audit assertion issuance** — Log every assertion issued