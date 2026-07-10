# OIDC Claims and Scopes: Reference and GGID Implementation Analysis

> Spec: [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)

## 1. Overview

OpenID Connect (OIDC) is an identity layer built on top of OAuth 2.0. While OAuth 2.0 handles **authorization** (delegated access to APIs), OIDC handles **authentication** (proving who the user is).

Three core concepts:

- **Claims** — assertions about the end-user (email, name, phone, etc.), delivered as key-value pairs in JSON.
- **Scopes** — coarse-grained tokens that request a *set* of claims (e.g., `email` grants `email` + `email_verified`).
- **ID Token** — a signed JWT returned alongside the access token, containing the requested claims directly.
- **UserInfo Endpoint** — a separate REST endpoint where an RP (Relying Party) can fetch claims using the access token.

The flow: an RP requests scopes at `/authorize`, receives an authorization code, exchanges it at `/token` for an access token + ID token, and can optionally call `/userinfo` for fresh claims.

## 2. Standard Scopes

| Scope | Claims Granted | Notes |
|-------|---------------|-------|
| `openid` | `sub` | REQUIRED for OIDC flows. Without it, the request is plain OAuth 2.0. |
| `profile` | `name`, `family_name`, `given_name`, `middle_name`, `nickname`, `preferred_username`, `profile`, `picture`, `website`, `gender`, `birthdate`, `zoneinfo`, `locale`, `updated_at` | Default human-readable profile information. |
| `email` | `email`, `email_verified` | End-user's email address and verification status. |
| `address` | `address` (JSON object) | `formatted`, `street_address`, `locality`, `region`, `postal_code`, `country`. |
| `phone` | `phone_number`, `phone_number_verified` | End-user's phone number and verification status. |
| `offline_access` | (grants refresh token) | Requires explicit consent — cannot be silent. Enables long-lived access via refresh tokens. |

```json
// Authorization request example
GET /oauth/authorize?
  response_type=code&
  client_id=gcid_abc123&
  redirect_uri=https://app.example.com/callback&
  scope=openid%20profile%20email%20offline_access&
  state=random_state&
  nonce=random_nonce
```

## 3. Standard Claims Reference

| Claim | Type | Scope | Description | Example |
|-------|------|-------|-------------|---------|
| `sub` | string | `openid` | Subject identifier (REQUIRED) | `"12345"` or pairwise hash |
| `name` | string | `profile` | Full display name | `"Alice Zhang"` |
| `given_name` | string | `profile` | First name | `"Alice"` |
| `family_name` | string | `profile` | Last name | `"Zhang"` |
| `middle_name` | string | `profile` | Middle name | `"M"` |
| `nickname` | string | `profile` | Casual name | `"Aly"` |
| `preferred_username` | string | `profile` | Login handle | `"alice.z"` |
| `profile` | string (URI) | `profile` | Profile page URL | `"https://example.com/alice"` |
| `picture` | string (URI) | `profile` | Avatar URL | `"https://example.com/alice.jpg"` |
| `website` | string (URI) | `profile` | Personal website | `"https://alice.dev"` |
| `email` | string | `email` | Email address | `"alice@example.com"` |
| `email_verified` | boolean | `email` | Email confirmed? | `true` |
| `gender` | string | `profile` | Gender | `"female"` |
| `birthdate` | string | `profile` | ISO 8601 date | `"1990-01-15"` |
| `zoneinfo` | string | `profile` | Time zone | `"America/New_York"` |
| `locale` | string | `profile` | BCP47 locale | `"en-US"` |
| `phone_number` | string | `phone` | E.164 phone | `"+15551234567"` |
| `phone_number_verified` | boolean | `phone` | Phone confirmed? | `true` |
| `address` | object | `address` | Postal address JSON | `{"formatted":"..."}` |
| `updated_at` | integer | `profile` | Last update (unix) | `1700000000` |

## 4. Subject Identifier Types

### Public (opaque)

Same `sub` value for all RPs. Simple and deterministic, but creates a privacy risk: multiple RPs can correlate the same user across services.

```
sub: "550e8400-e29b-41d4-a716-446655440000"  // same for every RP
```

### Pairwise

Different `sub` value per RP, derived from the user ID + the RP's sector identifier. This prevents cross-RP correlation.

**Algorithm:**
```
sub = base64url(SHA256(sector_identifier + user_id + pepper))
```

Example: user 123 at `app1.com` → `"abc123def456"`, same user at `app2.com` → `"xyz789ghi012"`.

Each RP declares its domain via `sector_identifier_uri`. GGID should support pairwise for privacy-sensitive deployments.

## 5. UserInfo Endpoint

### Purpose

A separate REST endpoint (`GET` or `POST /userinfo`) where an RP can fetch claims after the initial ID token. Useful when claims may change (profile update, group membership change).

**Authorization:** `Bearer <access_token>` with `openid` scope.

### Response

```json
// 200 OK — Content-Type: application/json
{
  "sub": "12345",
  "name": "Alice Zhang",
  "email": "alice@example.com",
  "email_verified": true,
  "picture": "https://example.com/alice.jpg",
  "updated_at": 1700000000
}
```

May also be a signed/encrypted JWT (Content-Type: `application/jwt`).

### Error Handling

| Status | Error | Cause |
|--------|-------|-------|
| 401 | `invalid_token` | Missing, expired, or malformed token |
| 403 | `insufficient_scope` | Token lacks `openid` scope |
| 400 | `invalid_request` | Malformed request parameters |

## 6. Claim Request Parameter

The `claims` authorization request parameter allows an RP to request **individual claims** with fine-grained control — more specific than scopes.

```json
// Request email in the ID token + name via UserInfo
{
  "id_token": {
    "email": null,
    "email_verified": null
  },
  "userinfo": {
    "name": null
  }
}
```

URL-encoded: `claims={"id_token":{"email":null}}`

### Voluntary vs Essential Claims

| Type | Syntax | Behavior |
|------|--------|----------|
| Voluntary | `"email": null` | RP wants it; auth proceeds even if unavailable |
| Essential | `"email": {"essential": true}` | RP requires it; fail if unavailable |
| Value constraint | `"acr": {"values": ["urn:mace:incommon:iap:silver"]}` | Require specific ACR values |
| Max age | `"auth_time": {"essential": true, "max_age": 600}` | Require recent authentication |

### Purpose Field

The `purpose` field is displayed on the consent screen to explain *why* a claim is needed:

```json
{
  "email": {"essential": true, "purpose": "Order confirmation"}
}
```

## 7. Aggregated and Distributed Claims

OIDC Core Section 5.6 defines two mechanisms for including claims from external sources.

### Aggregated Claims

Claims from external providers embedded directly in the ID token or UserInfo response. The token includes `_claim_sources` and `_claim_names` metadata.

```json
{
  "sub": "12345",
  "credit_score": 720,
  "_claim_names": {
    "credit_score": "src1"
  },
  "_claim_sources": {
    "src1": {
      "JWT": "eyJhbGci..."
    }
  }
}
```

### Distributed Claims

A reference to an external endpoint the RP must fetch using a provided access token.

```json
{
  "sub": "12345",
  "_claim_names": {
    "address": "src1"
  },
  "_claim_sources": {
    "src1": {
      "endpoint": "https://address-verify.example.com/claims",
      "access_token": "xyz789"
    }
  }
}
```

### Use Cases

- **Federated identity**: combine claims from multiple IdPs
- **Progressive profiling**: add claims over time without re-authentication
- **Third-party verification**: verified email/phone/address from external services

## 8. GGID Implementation Analysis

Examining `services/oauth/internal/service/oauth_service.go` and `services/oauth/internal/server/server.go`:

**Current state:**

- **ID Token** (`issueIDToken`): Contains `iss`, `sub`, `aud`, `iat`, `exp`, `nonce`, `tenant_id` + optional `amr`, `acr`, `auth_time`. Does NOT populate `profile`/`email`/`address`/`phone` claims.
- **UserInfo** (`GetUserInfo`): Returns `sub`, `name`, `email`, `email_verified`, `picture`, `tenant_id` from access token claims — but the access token itself only contains `sub` and `tenant_id`, so `name`, `email`, etc. are always empty.
- **Discovery** advertises `scopes_supported`: `openid`, `profile`, `email`, `offline_access` and `claims_supported`: `sub`, `email`, `name`, `picture`, `groups`, `preferred_username`, `updated_at`.
- **Subject type**: `public` only (`SubjectTypesSupported: ["public"]`).
- **Consent**: Basic consent flow for non-standard scopes exists (`/oauth/consent` endpoint).
- **Refresh tokens**: Supported via `offline_access` with rotation + reuse detection.

| Feature | Current State | Gap |
|---------|--------------|-----|
| `openid` scope + `sub` claim | Implemented in ID token | Profile/email claims missing from ID token |
| `profile` scope | Advertised in discovery | Claims not populated in ID token or UserInfo |
| `email` scope | Advertised in discovery | Claims not populated |
| `address` / `phone` scopes | Not advertised | Missing entirely |
| `offline_access` | Implemented (refresh token rotation) | None |
| UserInfo endpoint | Endpoint exists | Returns empty claims (no user data lookup) |
| `claims` request parameter | Not implemented | Missing |
| Pairwise subject identifiers | Not implemented | Missing |
| Aggregated/distributed claims | Not implemented | Missing |
| Consent for scopes | Basic (boolean check) | No per-scope granular consent |
| ID token `amr`/`acr`/`auth_time` | Implemented (`IDTokenOptions`) | Correct |

**Root cause**: The `issueIDToken` and `issueAccessToken` methods only receive `userID` and `tenantID` — they never fetch user profile data from the identity service or database. The `GetUserInfo` method parses claims from the access token JWT rather than looking up the user record.

## 9. Roadmap

| Phase | Deliverable | Effort | Priority |
|-------|------------|--------|----------|
| 1 | Populate standard claims (`profile`, `email`) in ID token by fetching user data | 2-3 days | High |
| 2 | Wire UserInfo endpoint to identity service (real user data lookup) | 1-2 days | High |
| 3 | Pairwise subject identifiers (`sector_identifier` + SHA256) | 2 days | Medium |
| 4 | `claims` request parameter parsing (voluntary + essential) | 3 days | Medium |
| 5 | `address` + `phone` scope support (requires identity schema) | 2 days | Medium |
| 6 | Aggregated/distributed claims | 5 days | Low (future) |

### Phase 1 sketch: enriching the ID token

```go
// After resolving userID, fetch user profile from identity service
func (s *OAuthService) issueIDToken(ctx context.Context, userID uuid.UUID, /* ... */) (string, error) {
    // ... existing registered claims ...
    user, err := s.identityClient.GetUser(ctx, userID)
    if err != nil {
        return "", fmt.Errorf("fetch user: %w", err)
    }
    if contains(scopes, "profile") {
        claims["name"] = user.DisplayName
        claims["preferred_username"] = user.Username
        claims["picture"] = user.AvatarURL
    }
    if contains(scopes, "email") {
        claims["email"] = user.Email
        claims["email_verified"] = user.EmailVerified
    }
    // ... sign and return ...
}
```

Phase 1-2 unblocks real OIDC conformance — without user data in tokens, the `profile` and `email` scopes are non-functional. Phase 3 (pairwise) is critical for privacy-sensitive multi-tenant deployments. Phases 4-5 bring GGID to full OIDC Core compliance. Phase 6 is advanced federation for enterprise customers.
