# OAuth 2.1 Compliance Statement

This guide documents GGID's OAuth 2.1 compliance matrix, implemented features, partial/gap features, compatibility notes, and upgrade path.

## Compliance Matrix

| Requirement | RFC | Status | Implementation |
|---|---|---|---|
| PKCE mandatory (S256) | 7636 | Compliant | Required for all clients |
| Exact redirect URI match | 6749 | Compliant | No wildcards, no prefix |
| Implicit grant removed | 2.1 draft | Compliant | Disabled by default |
| Password grant removed | 2.1 draft | Compliant | Never implemented |
| Refresh token rotation | 6749 | Compliant | One-time use + reuse detection |
| State parameter required | 6749 | Compliant | CSRF protection enforced |
| Token introspection | 7662 | Compliant | POST /oauth/introspect |
| Token revocation | 7009 | Compliant | POST /oauth/revoke |
| PAR (Pushed Auth Request) | 9126 | Compliant | POST /oauth/par |
| JAR (JWT Auth Request) | 9101 | Compliant | Signed request objects |
| DPoP | 9449 | Compliant | cnf.jkt binding |
| mTLS client auth | 8705 | Compliant | tls_client_auth |
| Device authorization | 8628 | Compliant | POST /device_authorization |
| CIBA | 9126 | Compliant | POST /bc-authorize |
| Dynamic registration | 7591 | Compliant | POST /oauth/register |
| Discovery | 8414 | Compliant | /.well-known/openid-configuration |
| PKCE method S256 only | 7636 | Compliant | "plain" rejected |
| HTTPS required | 2.1 draft | Compliant | No HTTP redirect URIs |
| No wildcard CORS | 2.1 draft | Compliant | Origin-specific |
| Client authentication | 2.1 draft | Compliant | 5 methods supported |

## Implemented Features Detail

### PKCE (RFC 7636)

- Required for ALL clients (confidential and public)
- Only S256 method accepted (plain rejected)
- code_verifier: 43-128 characters
- code_challenge: BASE64URL(SHA256(code_verifier))

### Refresh Token Rotation

- One-time use: each refresh token consumed after exchange
- Reuse detection: using old token revokes entire family
- Family revocation: all tokens from same login session revoked
- Bounded lifetime: 7 days default, 30 days max

### DPoP (RFC 9449)

- DPoP proof: JWT signed with client's private key
- Token binding: cnf.jkt claim in access token
- Replay prevention: jti + htm + htu validation
- Resource server: validates DPoP proof + cnf match

### mTLS (RFC 8705)

- tls_client_auth: client cert during TLS handshake
- Cert fingerprint in client registration
- Self_signed_tls_client_auth: supported for testing

### PAR (RFC 9126)

- POST /oauth/par: push auth request as JWT
- Returns request_uri for authorize endpoint
- 60 second TTL on request_uri
- Signed request objects supported

### JAR (RFC 9101)

- request parameter: JWT-encoded auth request
- request_uri: reference to stored request object
- RS256/ES256 signing algorithms
- exp + jti for request validity and replay prevention

### CIBA (RFC 9126)

- POST /bc-authorize: backchannel auth request
- auth_req_id with 10 minute TTL
- Polling interval: 5 seconds
- Binding message support
- user_code support

## Compatibility Notes

### Backward Compatibility

| Feature | OAuth 2.0 | OAuth 2.1 | Migration |
|---|---|---|---|
| Implicit grant | Supported | Removed | Migrate to auth code + PKCE |
| Password grant | Supported | Removed | Migrate to auth code or device code |
| PKCE optional | Optional | Mandatory | Add PKCE to all clients |
| Redirect URI prefix | Some allowed | Exact only | Update registered URIs |

### Client Impact

- Public clients (SPAs): Must use PKCE (most already do)
- Confidential clients: Must add PKCE (new requirement)
- Legacy implicit clients: Must migrate to auth code flow
- Password grant clients: Must migrate to device code or auth code

## Upgrade Path

### Phase 1: PKCE Enforcement (Done)

```yaml
oauth:
  pkce:
    required: true
    method: "S256"
```

### Phase 2: Deprecate Implicit (Done)

```yaml
oauth:
  grants:
    implicit:
      enabled: false
```

### Phase 3: Refresh Rotation (Done)

```yaml
oauth:
  refresh_token:
    rotation: "required"
    reuse_detection: true
```

### Phase 4: Full 2.1 Compliance (Done)

```yaml
oauth:
  version: "2.1"
  enforce:
    pkce: true
    exact_redirect_match: true
    state_required: true
    no_implicit: true
    no_password: true
    refresh_rotation: true
```

## Discovery Endpoint Compliance

```json
{
  "issuer": "https://auth.ggid.example.com",
  "grant_types_supported": [
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code",
    "urn:ietf:params:oauth:grant-type:token-exchange",
    "urn:openid:params:grant-type:ciba"
  ],
  "response_types_supported": ["code", "code id_token"],
  "code_challenge_methods_supported": ["S256"],
  "require_pkce": true,
  "pushed_authorization_request_endpoint": "https://auth.ggid.example.com/oauth/par",
  "require_pushed_authorization_requests": false,
  "request_parameter_supported": true,
  "request_uri_parameter_supported": true,
  "backchannel_authentication_endpoint": "https://auth.ggid.example.com/oauth/bc-authorize",
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "private_key_jwt",
    "tls_client_auth"
  ],
  "revocation_endpoint": "https://auth.ggid.example.com/oauth/revoke",
  "introspection_endpoint": "https://auth.ggid.example.com/oauth/introspect"
}
```

## Best Practices

1. **Stay current with draft** — Monitor OAuth 2.1 draft changes
2. **Test all flows** — Automated tests for each grant type
3. **Document compliance** — Keep this statement updated
4. **Help clients migrate** — Provide migration guides
5. **Monitor for deprecated usage** — Log when deprecated features are attempted
6. **Participate in IETF** — Contribute to the standard
7. **Audit compliance annually** — Verify all requirements still met
8. **Publish discovery metadata** — Let clients auto-discover capabilities
9. **Support backward compat** — Don't break existing clients abruptly
10. **Plan for future RFCs** — Track new specifications