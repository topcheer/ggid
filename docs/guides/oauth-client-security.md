# OAuth Client Security

This guide covers client authentication methods, client secret management, redirect URI security, PKCE for all clients, token storage, and GGID's client security enforcement.

## Client Authentication Methods

### Method Comparison

| Method | Type | Security | Use Case |
|---|---|---|---|
| client_secret_basic | Confidential | Medium | Server-side apps |
| client_secret_post | Confidential | Medium | Legacy compatibility |
| private_key_jwt | Confidential | High | Enterprise apps |
| mTLS (RFC 8705) | Confidential | Very High | Service-to-service |
| none | Public | Low (PKCE compensates) | SPAs, mobile apps |

### client_secret_basic

```
POST /token
Authorization: Basic base64(client_id:client_secret)
grant_type=authorization_code&code=...&code_verifier=...
```

### private_key_jwt

```
POST /token
client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
client_assertion=<signed_jwt>
grant_type=authorization_code&code=...
```

The JWT assertion contains iss, sub, aud, jti, exp, iat.

### mTLS (RFC 8705)

Client presents certificate during TLS handshake. No shared secret needed.

```yaml
oauth:
  client_auth:
    mtls:
      enabled: true
      cert_subject_cn: "client.example.com"
```

### none (Public Clients)

Public clients (SPAs, mobile) use PKCE instead of secrets:

```
POST /token
grant_type=authorization_code&code=...&code_verifier=...&client_id=...
```

## Client Secret Management

### Generation

```go
func generateClientSecret() string {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    return "gcs_" + base64.RawURLEncoding.EncodeToString(bytes)
}
```

### Storage (Hashed)

```go
func hashClientSecret(secret string) string {
    return argon2id.Hash(secret, "client-secret-salt")
}
```

### Rotation

```yaml
oauth:
  client_secrets:
    rotation:
      enabled: true
      interval: 90d
      overlap: 7d
      notify_owner: true
```

### Rotation Flow

1. Generate new secret
2. Store both old and new (both valid during overlap)
3. Notify client owner: "Rotate secret by date"
4. After overlap period, invalidate old secret
5. Log rotation event

## Redirect URI Security

### Rules

1. **Exact match only** — No wildcard, no prefix matching
2. **HTTPS required** — Except localhost for development
3. **No query parameters** — Pre-registered URIs can't have query strings
4. **No fragment** — No #fragment in redirect URI
5. **One per client** — Each client has its own redirect URIs

### Validation

```go
func validateRedirectURI(client *Client, requested string) error {
    found := false
    for _, registered := range client.RedirectURIs {
        if registered == requested { found = true; break }
    }
    if !found { return ErrRedirectURIMismatch }

    u, _ := url.Parse(requested)
    if u.Scheme != "https" && u.Host != "localhost" && !isCustomScheme(u.Scheme) {
        return ErrRedirectURINotHTTPS
    }
    if u.RawQuery != "" { return ErrRedirectURIHasQuery }
    if u.Fragment != "" { return ErrRedirectURIHasFragment }
    return nil
}
```

### Common Attacks

| Attack | Description | Prevention |
|---|---|---|
| Open redirect | Attacker uses malicious redirect | Exact match |
| Redirect interception | Attacker intercepts redirect | PKCE |
| Redirect manipulation | Attacker modifies redirect URL | Exact match + HTTPS |
| Loopback abuse | Using http://localhost on shared machine | Unique port per client |

## PKCE for All Clients

### Why PKCE for Confidential Clients?

OAuth 2.1 mandates PKCE for ALL clients. PKCE prevents:
- Authorization code interception
- Code injection attacks
- Mix-up attacks

### PKCE Flow

```
1. Client generates code_verifier (random 43-128 chars)
2. code_challenge = BASE64URL(SHA256(code_verifier))
3. Client -> /authorize?code_challenge=...&code_challenge_method=S256
4. Server stores code_challenge
5. Client -> /token?code_verifier=...
6. Server verifies: SHA256(code_verifier) == stored code_challenge
```

### Enforcement

```yaml
oauth:
  pkce:
    required: true
    method: "S256"
    verifier_length: 43
```

```go
func validatePKCE(codeVerifier, storedChallenge, method string) error {
    if codeVerifier == "" { return ErrMissingCodeVerifier }
    switch method {
    case "S256":
        hash := sha256.Sum256([]byte(codeVerifier))
        challenge := base64.RawURLEncoding.EncodeToString(hash[:])
        if challenge != storedChallenge { return ErrPKCEVerificationFailed }
    default:
        return ErrUnsupportedPKCEMethod
    }
    return nil
}
```

## Token Storage

### Web Application

| Storage | Security | Recommendation |
|---|---|---|
| httpOnly cookie | Good | Recommended for web apps |
| In-memory (JS) | Medium | Lost on page refresh |
| sessionStorage | Medium | Survives refresh |
| localStorage | Poor | XSS accessible |

### SPA (Public Client)

```javascript
// Recommended: In-memory + silent refresh
let accessToken = null;
async function getToken() {
    if (!accessToken) {
        const result = await fetchSilentToken(); // iframe + prompt=none
        accessToken = result.access_token;
    }
    return accessToken;
}
```

### Mobile App

| Storage | Security | Recommendation |
|---|---|---|
| Keychain (iOS) | High | Recommended |
| Keystore (Android) | High | Recommended |
| Encrypted SharedPreferences | Medium | Acceptable |
| Plain SharedPreferences | Low | Never use |

### Server-to-Server

| Storage | Security | Recommendation |
|---|---|---|
| Environment variables | Medium | Acceptable for containers |
| Secret manager (Vault/KMS) | High | Recommended |
| Config file | Low | Never use |

## GGID Client Security Enforcement

### Client Registration

```bash
POST /oauth/register
{
  "client_name": "Web Application",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "redirect_uris": ["https://app.example.com/callback"],
  "token_endpoint_auth_method": "private_key_jwt",
  "scope": "openid profile email",
  "pkce_required": true
}
```

### Security Enforcement

```yaml
oauth:
  client_security:
    pkce:
      required: true
      method: "S256"
    redirect_uri:
      exact_match: true
      require_https: true
    client_auth:
      methods: ["client_secret_basic", "private_key_jwt", "tls_client_auth"]
    secrets:
      min_length: 32
      rotation_interval: 90d
      hash_at_rest: true
```

### Per-Client Configuration

```yaml
clients:
  - id: "web-app"
    type: "confidential"
    auth_method: "private_key_jwt"
    pkce: true
    redirect_uris: ["https://app.example.com/callback"]
    secret_rotation: 90d

  - id: "mobile-app"
    type: "public"
    auth_method: "none"
    pkce: true
    redirect_uris: ["myapp://callback"]

  - id: "service-api"
    type: "confidential"
    auth_method: "tls_client_auth"
    grant_types: ["client_credentials"]
    mtls_cert: "sha256/cert-hash"
```

## Best Practices

1. **PKCE for all clients** — Not just public clients
2. **Exact redirect URI match** — No wildcards, no prefix matching
3. **Use private_key_jwt or mTLS** — Stronger than client_secret
4. **Hash secrets at rest** — Argon2id, never plaintext
5. **Rotate secrets regularly** — 90 days with overlap period
6. **Use Keychain/Keystore on mobile** — Never plain SharedPreferences
7. **Use httpOnly cookies for web** — Don't store tokens in localStorage
8. **Reject deprecated methods** — No implicit, no password grant
9. **Rate limit token endpoint** — Prevent brute force on secrets
10. **Audit client authentication** — Log all client auth attempts