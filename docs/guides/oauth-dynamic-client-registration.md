# OAuth Dynamic Client Registration (RFC 7591/7592)

Registration endpoint, client metadata, update/delete, software statements, and rate limits.

## Overview

Dynamic Client Registration (DCR) allows OAuth clients to register programmatically without manual admin setup. RFC 7591 defines registration; RFC 7592 defines management (read/update/delete).

## Registration Endpoint

### Create Client (RFC 7591)

```bash
POST /api/v1/oauth/register
Content-Type: application/json

{
  "client_name": "My Analytics App",
  "redirect_uris": [
    "https://analytics.example.com/callback"
  ],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid profile email users:read",
  "logo_uri": "https://analytics.example.com/logo.png",
  "policy_uri": "https://analytics.example.com/privacy",
  "tos_uri": "https://analytics.example.com/terms"
}
# → 201 Created
# {
#   "client_id": "client_abc123",
#   "client_secret": "secret_xyz789",
#   "client_id_issued_at": 1700000000,
#   "client_secret_expires_at": 0,
#   "client_name": "My Analytics App",
#   "redirect_uris": [...],
#   ...
# }
```

### Registration Access Levels

| Level | Auth Required | Scopes Allowed |
|-------|--------------|----------------|
| Open | None | `openid`, `profile`, `email` |
| Authenticated | Bearer token | + `users:read`, `roles:read` |
| Pre-authorized | Software statement | + `users:write`, `roles:assign` |

## Client Metadata Fields

| Field | Required | Description |
|-------|----------|-------------|
| `client_name` | Yes | Human-readable name |
| `redirect_uris` | Yes | Array of allowed callback URLs |
| `grant_types` | Yes | Supported grant types |
| `response_types` | Yes | Supported response types |
| `token_endpoint_auth_method` | No | `client_secret_basic` (default), `private_key_jwt`, `none` (PKCE) |
| `scope` | No | Requested scopes (within allowed set) |
| `contacts` | No | Admin email addresses |
| `logo_uri` | No | Client logo URL |
| `policy_uri` | No | Privacy policy URL |
| `tos_uri` | No | Terms of service URL |
| `jwks_uri` | No | Client JWKS URL (for `private_key_jwt`) |
| `software_id` | No | UUID identifying the software product |
| `software_version` | No | Version of the software |

## Manage Client (RFC 7592)

### Read Configuration

```bash
GET /api/v1/oauth/register/{client_id}
Authorization: Bearer <registration_access_token>
# → Full client metadata
```

### Update Configuration

```bash
PUT /api/v1/oauth/register/{client_id}
Authorization: Bearer <registration_access_token>
Content-Type: application/json

{
  "client_name": "My Analytics App v2",
  "redirect_uris": [
    "https://analytics.example.com/callback",
    "https://analytics.example.com/v2/callback"
  ],
  "scope": "openid profile email users:read users:write"
}
# → 200 with updated metadata + new registration_access_token
```

### Delete Client

```bash
DELETE /api/v1/oauth/register/{client_id}
Authorization: Bearer <registration_access_token>
# → 204 No Content
# All tokens issued to this client are immediately revoked
```

## Software Statement

A software statement is a signed JWT from a trusted third party that pre-validates client metadata. This allows higher scopes without admin approval.

```bash
POST /api/v1/oauth/register
{
  "software_statement": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "client_name": "Verified App",
  "redirect_uris": ["https://app.example.com/callback"]
}
```

### Software Statement JWT

```json
{
  "iss": "trusted-marketplace.com",
  "sub": "software-uuid",
  "aud": "https://auth.ggid.dev",
  "exp": 1700000000,
  "client_name": "Verified App",
  "redirect_uris": ["https://app.example.com/callback"],
  "scope": "openid profile users:read users:write",
  "software_id": "uuid",
  "software_version": "1.2.0"
}
```

### Trusted Issuers

```bash
# Admin configures trusted software statement issuers
POST /api/v1/admin/oauth/trusted-issuers
{
  "issuer": "trusted-marketplace.com",
  "jwks_uri": "https://trusted-marketplace.com/.well-known/jwks.json",
  "allowed_scopes": ["openid", "profile", "email", "users:read", "users:write"]
}
```

## Registration Access Token (RAT)

The RAT is a special token that allows managing the registered client. It is:
- Issued only once at registration time
- Bound to a specific `client_id`
- Required for all RFC 7592 operations (GET/PUT/DELETE)
- Revoked when the client is deleted or secret rotated

```json
{
  "registration_access_token": "rat_xyz...",
  "client_id": "client_abc123"
}
```

## Security Controls

### Redirect URI Validation

- Must be HTTPS (except `localhost`)
- No wildcards — exact match only
- Fragment (`#`) not allowed
- IP addresses blocked (except `127.0.0.1`)
- Maximum 10 redirect URIs per client

### Rate Limiting

| Action | Rate | Burst |
|--------|------|-------|
| Registration | 10/hour per IP | 20 |
| Read (GET) | 60/min | 120 |
| Update (PUT) | 20/min | 40 |
| Delete (DELETE) | 5/hour | 10 |

### Allowed Scopes by Registration Type

| Scope | Open | Authenticated | Software Statement |
|-------|------|--------------|-------------------|
| `openid` | ✅ | ✅ | ✅ |
| `profile` | ✅ | ✅ | ✅ |
| `email` | ✅ | ✅ | ✅ |
| `users:read` | ❌ | ✅ | ✅ |
| `users:write` | ❌ | ❌ | ✅ |
| `*:admin` | ❌ | ❌ | ❌ (CISO only) |

## Error Handling

```json
{
  "error": "invalid_redirect_uri",
  "error_description": "Redirect URI must use HTTPS (except localhost)"
}
```

| Error | Cause |
|-------|-------|
| `invalid_client_metadata` | Missing required field or invalid value |
| `invalid_redirect_uri` | Non-HTTPS, wildcard, or blocked |
| `invalid_token` | RAT missing, expired, or wrong client |
| `access_denied` | Scope not allowed at registration level |
| `invalid_software_statement` | Bad signature, expired, or untrusted issuer |

## Monitoring

| Metric | Alert |
|--------|-------|
| Registration spike | >50/hour from single IP → possible abuse |
| Unusual redirect URIs | New domain not seen before |
| Software statement failures | >5% → check issuer JWKS |
| Client never used after 30 days | Flag for cleanup |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [OAuth Scope Design](oauth-scope-design.md)
