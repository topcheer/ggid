# OpenID Connect Advanced

This guide covers advanced OIDC flows: hybrid flow, refresh token rotation, DPoP with OIDC, pairwise vs public subjects, prompt=none silent auth, claims parameter (RFC 9396), JAR (RFC 9101), CIBA (RFC 9126), and multiple response types.

## Hybrid Flow

Hybrid flow combines authorization code and implicit flows, returning some tokens from the authorization endpoint and others from the token endpoint.

### Response Types

| response_type | Authorization Endpoint Returns | Token Endpoint Returns |
|---|---|---|
| `code id_token` | Code + ID Token | Access + Refresh |
| `code token` | Code + Access Token | Access + Refresh |
| `code id_token token` | Code + ID + Access | Access + Refresh |

### Flow

```
1. Client → GET /authorize?response_type=code%20id_token&client_id=...&nonce=...
2. Server returns: code + id_token (front-channel)
3. ID Token contains c_hash (code hash) for binding
4. Client → POST /token (grant_type=authorization_code, code)
5. Server returns: access_token + refresh_token
```

### Security: c_hash and at_hash

The ID Token from the front-channel includes hash bindings:

```json
{
  "c_hash": "hash_of_authorization_code",
  "at_hash": "hash_of_access_token"
}
```

These prevent token substitution attacks by binding the front-channel tokens to the back-channel tokens.

### When to Use Hybrid Flow

| Scenario | Recommended Flow |
|---|---|
| Native mobile app | Authorization Code + PKCE |
| SPA | Authorization Code + PKCE |
| Legacy client needing front-channel ID Token | Hybrid (`code id_token`) |
| Maximum security | Hybrid (`code id_token token`) |

## Refresh Token Rotation in OIDC

### OIDC Refresh Token Extensions

OIDC adds `offline_access` scope for refresh token issuance:

```
GET /authorize?scope=openid%20offline_access&...
```

### Rotation with OpenID Connect Session Management

```json
{
  "access_token": "eyJ...",
  "refresh_token": "rt-new-001",
  "id_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_expires_in": 604800
}
```

### Session Management (RFC 7836/OIDC Session)

```
GET /authorize?prompt=none&session_state=...
```

Checks if the user's session at the IdP is still valid without UI interaction.

## DPoP with OIDC

### ID Token Binding

When DPoP is used with OIDC, the ID Token includes confirmation claim:

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid",
  "aud": "client-id",
  "cnf": {
    "jkt": "sha256-hash-of-dpop-public-key"
  }
}
```

### Flow

```
1. Client generates DPoP key pair
2. Client → /authorize with DPoP header (Authorization Request)
3. Server validates DPoP proof
4. Client → /token with DPoP header (Token Request)
5. Server issues DPoP-bound tokens (cnf claim)
6. Resource Server validates DPoP proof + cnf match
```

### Benefits

- ID Token is bound to the client's key
- Prevents token replay even if intercepted
- No TLS client certificate needed

## Pairwise vs Public Subjects

### Public Subject (sub)

Same `sub` value across all clients:

```json
{ "sub": "user-uuid-1234" }
```

**Risk**: Clients can correlate users by comparing `sub` values.

### Pairwise Subject

Different `sub` value per client:

```
Client A: sub = "pairwise-hash-A-5678"
Client B: sub = "pairwise-hash-B-9012"
```

**Generation**: `sub = SHA256(sector_identifier + user_id + pairwise_salt)`

### Sector Identifiers

For clients under the same organization, a sector identifier URI groups them to share the same pairwise sub:

```yaml
oidc:
  subject_types_supported:
    - "public"
    - "pairwise"
  pairwise:
    salt: "<per-tenant-random-salt>"
    sector_identifier_uri: "https://app.example.com/sector"
```

### Configuration

```yaml
oidc:
  subject_type: "pairwise"  # or "public"
  pairwise:
    salt: "<32-byte-random-salt>"
    sector_identifiers:
      "app.example.com": "https://app.example.com/sector"
```

## prompt=none Silent Authentication

### How It Works

```
GET /authorize?prompt=none&response_type=code&client_id=...
```

The server checks if the user has an existing session:
- **Session valid**: Returns authorization code immediately (no UI)
- **Session expired**: Returns `error=login_required` (no UI)

### Use Cases

| Use Case | Description |
|---|---|
| Single sign-on (SSO) | Check if user is already logged in |
| Session refresh | Get new tokens without re-authentication |
| Embedded apps | iframe-based session check |

### prompt Values

| Value | Behavior |
|---|---|
| `none` | No UI — return error if interaction needed |
| `login` | Force re-authentication |
| `consent` | Force consent screen |
| `select_account` | Force account selection |

### Error Responses

| Error | Meaning |
|---|---|
| `login_required` | No session, prompt=none specified |
| `consent_required` | Consent needed, prompt=none specified |
| `interaction_required` | Some interaction needed |
| `account_selection_required` | Account selection needed |

## Claims Parameter (RFC 9396)

### Requesting Specific Claims

```
GET /authorize?claims={"id_token":{"email":{"essential":true},"email_verified":null},"userinfo":{"name":null,"picture":null}}
```

### Claim Specification

```json
{
  "id_token": {
    "email": {"essential": true},
    "email_verified": null,
    "acr": {"essential": true, "values": ["urn:mace:incommon:iap:silver"]}
  },
  "userinfo": {
    "name": null,
    "picture": null,
    "department": {"essential": false}
  }
}
```

| Property | Description |
|---|---|
| `null` | Request claim but don't require it |
| `essential: true` | Claim must be present or request fails |
| `values: [...]` | Claim must have one of these values |
| `value: "..."` | Claim must have this exact value |

### Voluntary vs Essential Claims

- **Voluntary** (`null`): Requested but not required
- **Essential** (`{"essential": true}`): Must be present, request fails if absent

### Use Cases

1. **Step-up auth**: `"acr": {"essential": true, "values": ["urn:...:silver"]}`
2. **Require email**: `"email": {"essential": true}`
3. **Request specific locale**: `"locale": {"value": "fr-FR"}`

## JAR (JWT-Secured Authorization Request, RFC 9101)

### What is JAR?

Authorization request parameters are wrapped in a signed JWT, allowing the request to be authenticated and integrity-protected.

### Request Object

```
GET /authorize?request=<JWT>&client_id=...
```

The JWT contains all authorization parameters:

```json
{
  "iss": "client-id",
  "aud": "https://auth.ggid.example.com",
  "response_type": "code",
  "client_id": "client-id",
  "redirect_uri": "https://app.example.com/cb",
  "scope": "openid profile email",
  "state": "xyz",
  "nonce": "abc",
  "code_challenge": "...",
  "code_challenge_method": "S256",
  "exp": 1700000600,
  "jti": "unique-request-id"
}
```

### JAR + Request URI

For large requests, use `request_uri`:

```
1. Client → POST /request (JWT)
2. Server stores JWT, returns request_uri
3. Client → GET /authorize?request_uri=https://auth.ggid.example.com/request/abc
4. Server fetches JWT from request_uri
```

### Benefits

- Request integrity (signed by client)
- Request authentication (client identity verified)
- Large request support (via request_uri)
- Reuse prevention (exp + jti)

### Configuration

```yaml
oidc:
  jar:
    enabled: true
    require_signed_requests: false  # Make mandatory for high-security
    allowed_algorithms: ["RS256", "ES256"]
    request_uri:
      enabled: true
      ttl: 60s  # Request URI valid for 60 seconds
```

## CIBA (Client-Initiated Backchannel Authentication, RFC 9126)

### What is CIBA?

Decouples authentication from the user's browser. The client requests authentication, and the user approves on a separate device (e.g., phone app).

### Flow

```
1. Client → POST /bc-authorize (client_id, scope, login_hint=binding_value)
2. Server returns: auth_req_id + expires_in + interval
3. User receives notification on their device → approves/denies
4. Client polls: POST /token (grant_type=urn:openid:params:grant-type:ciba, auth_req_id)
5. Server returns: authorization_pending → authorization_pending → access_token
6. Or: POST /token with auth_req_id → access_token (if user already approved)
```

### Backchannel Authorization Request

```bash
POST /bc-authorize
Content-Type: application/x-www-form-urlencoded

grant_type=urn:openid:params:grant-type:ciba-authentication
client_id=client-id
client_secret=secret
scope=openid profile email
login_hint=user@example.com
binding_message=Login to Web App?
user_code=1234
```

### Response

```json
{
  "auth_req_id": "ciba-req-uuid",
  "expires_in": 600,
  "interval": 5
}
```

### Polling

```bash
POST /token
grant_type=urn:openid:params:grant-type:ciba
auth_req_id=ciba-req-uuid
```

| Response | Meaning |
|---|---|
| `authorization_pending` | User hasn't responded yet |
| `slow_down` | Polling too fast |
| `access_token` + `id_token` | User approved |
| `error: access_denied` | User denied |
| `error: expired_token` | auth_req_id expired |

### Binding Message

A message displayed on both the client and the user's device for verification:

```
Client displays: "Approve login on your phone. Code: 4915"
Phone shows: "Login to Web App? Code: 4915"
```

### Configuration

```yaml
oidc:
  ciba:
    enabled: true
    auth_req_id_lifetime: 600  # 10 minutes
    polling_interval: 5  # seconds
    max_polling_interval: 30
    binding_message:
      required: true
      max_length: 20
    user_code:
      optional: true
      length: 4
```

## Multiple Response Types

### Supported Combinations

| response_type | Flow |
|---|---|
| `code` | Authorization Code |
| `id_token` | Implicit (ID Token only) |
| `token` | Implicit (Access Token only) |
| `id_token token` | Implicit (ID + Access) |
| `code id_token` | Hybrid |
| `code token` | Hybrid |
| `code id_token token` | Hybrid (all) |

### Configuration

```yaml
oidc:
  response_types_supported:
    - "code"
    - "code id_token"
    - "code token"
    - "code id_token token"
  # Note: Implicit types (token, id_token) deprecated in OAuth 2.1
```

## GGID Implementation

### Discovery Endpoint

```bash
GET /.well-known/openid-configuration

{
  "issuer": "https://auth.ggid.example.com",
  "authorization_endpoint": "https://auth.ggid.example.com/oauth/authorize",
  "token_endpoint": "https://auth.ggid.example.com/oauth/token",
  "userinfo_endpoint": "https://auth.ggid.example.com/oauth/userinfo",
  "bc_authorize_endpoint": "https://auth.ggid.example.com/oauth/bc-authorize",
  "introspection_endpoint": "https://auth.ggid.example.com/oauth/introspect",
  "revocation_endpoint": "https://auth.ggid.example.com/oauth/revoke",
  "jwks_uri": "https://auth.ggid.example.com/.well-known/jwks.json",
  "response_types_supported": ["code", "code id_token", "code token", "code id_token token"],
  "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials", "urn:openid:params:grant-type:ciba"],
  "subject_types_supported": ["public", "pairwise"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256", "EdDSA"],
  "scopes_supported": ["openid", "profile", "email", "address", "phone", "offline_access"],
  "claims_supported": ["sub", "name", "email", "email_verified", "roles", "groups", "tenant_id"],
  "request_parameter_supported": true,
  "request_uri_parameter_supported": true,
  "require_request_uri_registration": false,
  "prompt_values_supported": ["none", "login", "consent", "select_account"]
}
```

## Best Practices

1. **Prefer authorization code + PKCE** — Don't use implicit flow
2. **Use pairwise subjects** — Prevent cross-client correlation
3. **Enable JAR for high-security** — Signed authorization requests
4. **Rate limit CIBA polling** — Prevent polling abuse
5. **Validate binding messages** — Prevent CIBA phishing
6. **Use prompt=none for SSO** — Silent session check
7. **Rotate refresh tokens** — One-time use with reuse detection
8. **Bind tokens with DPoP** — Prevent token replay
9. **Document supported response types** — Clear discovery metadata
10. **Test all flows** — Automated tests for each response type