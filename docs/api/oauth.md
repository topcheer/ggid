# OAuth Service API Reference

Complete REST API reference for GGID's OAuth 2.0 / OIDC service.

**Base URL**: `https://api.ggid.example.com`

## Authorization

```
GET /oauth/authorize?response_type=code&client_id=xxx&redirect_uri=xxx&scope=openid+profile&state=xxx&code_challenge=xxx&code_challenge_method=S256
```
**Response**: 302 redirect with `code` and `state`.

## Token

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
```

**Grants**:
| Grant | Parameters |
|-------|------------|
| `authorization_code` | code, redirect_uri, client_id, code_verifier |
| `client_credentials` | client_id, client_secret, scope |
| `refresh_token` | refresh_token, client_id |

**Response** (200):
```json
{"access_token":"eyJ...","token_type":"Bearer","expires_in":900,"refresh_token":"eyJ...","scope":"openid profile"}
```

**Errors**:
| Error | Status | Description |
|-------|--------|-------------|
| `invalid_request` | 400 | Missing parameter |
| `invalid_client` | 401 | Client auth failed |
| `invalid_grant` | 400 | Invalid/expired code |

## JWKS

```
GET /.well-known/jwks.json
```
```json
{"keys":[{"kty":"RSA","kid":"key-1","use":"sig","alg":"RS256","n":"...","e":"AQAB"}]}
```

## UserInfo

```
GET /oauth/userinfo
Authorization: Bearer <access_token>
```
```json
{"sub":"user-uuid","email":"alice@example.com","name":"Alice Chen","tenant_id":"tenant-uuid"}
```

## Introspection

```
POST /oauth/introspect
```
Requires client authentication.
```json
{"active":true,"scope":"users:read","client_id":"xxx","exp":1706105100,"sub":"user-uuid"}
```

## Revocation

```
POST /oauth/revoke
token=xxx&client_id=xxx&client_secret=xxx
```

## Discovery

```
GET /.well-known/openid-configuration
```
Returns issuer, endpoints, supported scopes/grants/algorithms.

## Dynamic Client Registration (RFC 7591)

```
POST /oauth/register
{"redirect_uris":["https://app.example.com/callback"],"grant_types":["authorization_code"]}
```
**Response** (201): `{"client_id":"...","client_secret":"..."}`

## Client Management

```
GET /api/v1/oauth/clients
POST /api/v1/oauth/clients
DELETE /api/v1/oauth/clients/{client_id}
```

## Consent

```
GET /api/v1/oauth/consent
DELETE /api/v1/oauth/consent/{client_id}
```

## PAR (RFC 9126) — Planned

```
POST /oauth/par
```
Stores auth request, returns `request_uri`.

## DPoP — Planned

``
Authorization: DPoP <token>
DPoP: <proof-jwt>
```

## JAR (RFC 9101)

Authorization request as JWT: `request` parameter or `request_uri`.

## CIBA (RFC 9126) — Planned

```
POST /oauth/bc-authorize
```
Client-initiated backchannel authentication.

## SAML Endpoints

```
GET /.well-known/saml-metadata
POST /saml/acs
GET /saml/slo
```

## See Also
- [OAuth API (detailed)](oauth-api.md)
- [OAuth 2.1 Migration](../guides/oauth-2-1-migration.md)
- [Auth API](auth.md)

```
