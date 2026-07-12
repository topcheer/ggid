# OAuth Advanced Security: PAR, JAR, and DPoP

Guide for Pushed Authorization Requests (RFC 9126), JWT-Secured Authorization Requests (RFC 9101), and Demonstrating Proof-of-Possession (RFC 7800/9449).

## When to Use Each

| Feature | Problem Solved | Best For | Overhead |
|---------|---------------|----------|----------|
| PAR | URL length + request tampering | Mobile, complex scopes | Extra round-trip |
| JAR | Request integrity + non-repudiation | High-security, B2B | JWT signing burden |
| DPoP | Bearer token theft replay | SPA, mobile, API access | Key management |
| PAR + JAR | Maximum security | Financial, healthcare | Both overheads |

## Pushed Authorization Requests (PAR)

### Problem

Standard OAuth authorization URLs can exceed browser URL length limits with many scopes and parameters. They're also visible in browser history and logs.

### Flow

```
Client → POST /par (request params as body) → Authorization Server
                                              → Returns request_uri
Client → GET /authorize?request_uri=URN → Authorization Server
                                              → Looks up saved request
                                              → Processes authorization
```

### Implementation

```bash
# Step 1: Push the authorization request
POST /api/v1/oauth/par
Content-Type: application/x-www-form-urlencoded

response_type=code
&client_id=client-123
&redirect_uri=https://app.example.com/callback
&scope=openid+profile+users:read+users:write
&state=xyz
&nonce=abc
&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
&code_challenge_method=S256

# → 201 Created
# {"request_uri": "urn:ietf:params:oauth:request_uri:abc123",
#  "expires_in": 60}

# Step 2: Redirect user with request_uri only
GET /api/v1/oauth/authorize?client_id=client-123&request_uri=urn:ietf:params:oauth:request_uri:abc123
```

### Per-Client Enforcement

```bash
# Require PAR for high-security clients
PATCH /api/v1/oauth/clients/{client_id}
{"require_par": true}
# Client must use PAR; direct /authorize with params is rejected
```

### Benefits

- No URL length limits
- Request parameters not in browser history/referrer
- Server validates request before user sees consent screen
- Prevents parameter tampering during redirect

## JWT-Secured Authorization Requests (JAR)

### Problem

Authorization request parameters can be tampered with in transit. JAR wraps the entire request in a signed JWT for integrity.

### Flow

```bash
# Step 1: Construct signed request object
# Header: {"alg":"ES256","kid":"key-1","typ":"JWT"}
# Payload:
{
  "response_type": "code",
  "client_id": "client-123",
  "redirect_uri": "https://app.example.com/callback",
  "scope": "openid profile users:read",
  "state": "xyz",
  "nonce": "abc",
  "max_age": 3600,
  "iss": "client-123",
  "aud": "https://auth.ggid.dev"
}

# Step 2: Send as request parameter
GET /api/v1/oauth/authorize?request=<signed-JWT>&client_id=client-123
```

### JAR + PAR Combined

```bash
# Maximum security: sign request object, then push via PAR
POST /api/v1/oauth/par
request=<signed-JWT>
# → request_uri returned
GET /api/v1/oauth/authorize?request_uri=<urn>
```

### JAR Requirements

- `iss` MUST equal `client_id`
- `aud` MUST be the authorization server's issuer URL
- `exp` MUST be set (max 60 minutes)
- Signing key must be registered in client's JWKS

## DPoP (Demonstrating Proof of Possession)

### Problem

Bearer tokens can be stolen and used by anyone. DPoP binds tokens to a private key held by the client, making stolen tokens useless without the key.

### Flow

```
1. Client generates DPoP key pair (stored locally)
2. Client signs each request with DPoP proof JWT
3. Server verifies proof: public key matches token's cnf claim
4. Without the private key, a stolen token is useless
```

### DPoP Proof JWT

```json
// Header
{"typ":"dpop+jwt","alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"...","y":"..."}}

// Payload
{
  "htu": "https://api.ggid.dev/v1/users",  // HTTP target URI
  "htm": "GET",                             // HTTP method
  "iat": 1700000000,                        // Issued at
  "jti": "nonce-uuid",                      // Unique per request
  "ath": "base64url(sha256(access_token))"  // Access token hash (for protected resources)
}
```

### Token Binding

```bash
# Token endpoint returns DPoP-bound token
POST /api/v1/oauth/token
DPoP: <dpop-proof-jwt>
grant_type=authorization_code&code=...&code_verifier=...

# Response includes cnf claim
{"access_token": "eyJ...", "cnf": {"jkt": "thumbprint-of-client-key"}}
```

The `cnf.jkt` (confirmation key thumbprint) binds the token to the client's DPoP key. Any API request must include a valid DPoP proof matching this thumbprint.

### Per-Client Enforcement

```bash
# Require DPoP for specific clients
PATCH /api/v1/oauth/clients/{client_id}
{"require_dpop": true}
# Tokens issued to this client will always include cnf binding
```

### DPoP vs mTLS

| Feature | DPoP | mTLS |
|---------|------|------|
| Key type | Asymmetric (EC/RSA) | X.509 certificate |
| Transport | Application layer | TLS layer |
| Mobile friendly | Yes | Complex cert mgmt |
| Browser/SPA | Yes (Web Crypto) | No |
| Server-to-server | Yes | Yes |
| Revocation | Rotate client key | Revoke certificate |

## Migration Strategy

```
Phase 1: Optional (current)
  → Clients opt into PAR/JAR/DPoP
  → Bearer tokens still accepted

Phase 2: Recommended
  → Console shows security recommendations
  → DPoP recommended for all SPA/mobile clients

Phase 3: Mandatory for high-risk
  → Financial/healthcare clients require PAR + DPoP
  → Bearer tokens deprecated for sensitive scopes
```

## Security Comparison

| Threat | Bearer Token | DPoP | mTLS |
|--------|-------------|------|------|
| Token theft from logs | Vulnerable | Protected | Protected |
| Token replay | Vulnerable | Protected (jti) | Protected |
| MITM token capture | Vulnerable | Protected | Protected |
| XSS token exfil | Vulnerable | Protected (key isolated) | N/A |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
- [OAuth Scope Design](oauth-scope-design.md)
