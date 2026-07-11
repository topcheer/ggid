# OAuth API Reference

Complete REST API reference for GGID's OAuth 2.0 / OIDC service.

**Base URL**: `https://api.ggid.example.com`

## Authorization

### Authorization Endpoint

```
GET /oauth/authorize
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `response_type` | Yes | `code` (implicit removed in OAuth 2.1) |
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must match registered URI exactly |
| `scope` | Yes | Requested scopes (space-separated) |
| `state` | Yes | CSRF protection (validated server-side) |
| `code_challenge` | Yes | PKCE challenge (S256) |
| `code_challenge_method` | Yes | `S256` |
| `nonce` | Recommended | Replay prevention for OIDC |

```bash
curl -v "https://api.ggid.example.com/oauth/authorize?response_type=code&client_id=xxx&redirect_uri=https://app.example.com/callback&scope=openid+profile&state=random123&code_challenge=xxx&code_challenge_method=S256"
```

**Response**: 302 redirect to `redirect_uri` with `code` and `state` parameters.

## Token

### Token Endpoint

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
```

**Authorization Code Grant**:
```bash
curl -X POST https://api.ggid.example.com/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "client_id=CLIENT_ID" \
  -d "client_secret=CLIENT_SECRET" \
  -d "code_verifier=PKCE_VERIFIER"
```

**Client Credentials Grant**:
```bash
curl -X POST https://api.ggid.example.com/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=CLIENT_ID" \
  -d "client_secret=CLIENT_SECRET" \
  -d "scope=users:read"
```

**Refresh Token Grant**:
```bash
curl -X POST https://api.ggid.example.com/oauth/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=REFRESH_TOKEN" \
  -d "client_id=CLIENT_ID"
```

**Response** (200):
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "eyJhbG...",
  "scope": "openid profile",
  "id_token": "eyJhbG..."
}
```

**Errors**:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `invalid_request` | 400 | Missing required parameter |
| `invalid_client` | 401 | Client authentication failed |
| `invalid_grant` | 400 | Invalid/expired code or refresh token |
| `invalid_scope` | 400 | Requested scope not allowed |
| `unauthorized_client` | 400 | Client not authorized for grant type |
| `unsupported_grant_type` | 400 | Grant type not supported |

## Introspection

```
POST /oauth/introspect
```

**Note**: Requires client authentication.

```bash
curl -X POST https://api.ggid.example.com/oauth/introspect \
  -u "client_id:client_secret" \
  -d "token=ACCESS_TOKEN"
```

**Response** (200):
```json
{
  "active": true,
  "scope": "users:read users:write",
  "client_id": "CLIENT_ID",
  "username": "alice@example.com",
  "exp": 1706105100,
  "iat": 1706104200,
  "sub": "user-uuid",
  "tenant_id": "tenant-uuid"
}
```

**Inactive token**:
```json
{ "active": false }
```

## JWKS

```
GET /.well-known/jwks.json
```

```bash
curl https://api.ggid.example.com/.well-known/jwks.json
```

**Response**:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-1",
      "use": "sig",
      "alg": "RS256",
      "n": "base64url-modulus",
      "e": "AQAB"
    }
  ]
}
```

## Discovery

```
GET /.well-known/openid-configuration
```

```bash
curl https://api.ggid.example.com/.well-known/openid-configuration
```

**Response**:
```json
{
  "issuer": "https://api.ggid.example.com",
  "authorization_endpoint": "https://api.ggid.example.com/oauth/authorize",
  "token_endpoint": "https://api.ggid.example.com/oauth/token",
  "introspection_endpoint": "https://api.ggid.example.com/oauth/introspect",
  "userinfo_endpoint": "https://api.ggid.example.com/oauth/userinfo",
  "jwks_uri": "https://api.ggid.example.com/.well-known/jwks.json",
  "registration_endpoint": "https://api.ggid.example.com/oauth/register",
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "client_credentials", "refresh_token"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256"],
  "code_challenge_methods_supported": ["S256"],
  "claims_supported": ["sub", "email", "name", "tenant_id", "scope"]
}
```

## Dynamic Client Registration (RFC 7591)

```
POST /oauth/register
```

```bash
curl -X POST https://api.ggid.example.com/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "redirect_uris": ["https://app.example.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email",
    "token_endpoint_auth_method": "client_secret_post"
  }'
```

**Response** (201):
```json
{
  "client_id": "generated-client-id",
  "client_secret": "generated-secret",
  "client_id_issued_at": 1706104200,
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"]
}
```

## Client Management

### List Clients

```
GET /api/v1/oauth/clients
```

### Create Client

```
POST /api/v1/oauth/clients
```

```bash
curl -X POST https://api.ggid.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "My Web App",
    "redirect_uris": ["https://app.example.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "scopes": ["openid", "profile", "users:read"]
  }'
```

### Delete Client

```
DELETE /api/v1/oauth/clients/{client_id}
```

## Consent

### Get User Consents

```
GET /api/v1/oauth/consent
```

### Revoke Consent

```
DELETE /api/v1/oauth/consent/{client_id}
```

## SAML Endpoints

### SP Metadata

```
GET /.well-known/saml-metadata
```

### SAML ACS

```
POST /saml/acs
```

### SAML SLO

```
GET /saml/slo
```

## See Also

- [REST API Reference](rest-api.md)
- [OAuth 2.1 Changes](../research/oauth-2.1-changes.md)
- [Authentication Flows](../guides/authentication-flows.md)
- [SAML Federation Guide](../guides/saml-federation-guide.md)
