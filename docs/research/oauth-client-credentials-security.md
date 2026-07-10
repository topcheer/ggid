# OAuth Client Credentials Flow Security for IAM Systems

**Document Type:** Security Research
**Scope:** OAuth 2.0 Client Credentials Grant (RFC 6749 §4.4) in IAM systems
**Relevant RFCs:** 6749, 7591, 7523, 8705, 9101
**Audit Target:** GGID OAuth Service (`services/oauth/`)

---

## Table of Contents

1. [Client Credentials Flow Overview](#1-client-credentials-flow-overview)
2. [Client Secret Storage](#2-client-secret-storage)
3. [mTLS-Bound Clients (RFC 8705)](#3-mtls-bound-clients-rfc-8705)
4. [JWT Assertion Auth (RFC 7523)](#4-jwt-assertion-auth-rfc-7523)
5. [Public vs Confidential Clients](#5-public-vs-confidential-clients)
6. [Client Authentication Methods Comparison](#6-client-authentication-methods-comparison)
7. [Client Credential Lifecycle](#7-client-credential-lifecycle)
8. [GGID Client Credentials Audit](#8-ggid-client-credentials-audit)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Client Credentials Flow Overview

The **client credentials grant** (RFC 6749 §4.4) is the standard OAuth 2.0
mechanism for machine-to-machine (M2M) authentication. Unlike authorization
code or implicit flows, no resource owner (user) participates — the client
application authenticates directly to the authorization server and receives
an access token.

### Flow Diagram

```
+----------+                           +------------------+
|  Client  |                           | Authorization     |
|  (M2M)   |                           | Server (GGID)     |
|          |---(A) client_id/secret--->|                  |
|          |    POST /oauth/token       |                  |
|          |    grant_type=             |                  |
|          |    client_credentials      |                  |
|          |<--(B) access_token---------|                  |
|          |                           |                  |
|          |---(C) access_token------->| Resource Server   |
|          |<--(D) protected data------|                   |
+----------+                           +------------------+
```

### Step Detail

**(A) Token Request:** The client sends an HTTP POST to the token endpoint:

```http
POST /oauth/token HTTP/1.1
Host: auth.example.com
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
&client_id=gcid_abc123def456
&client_secret=gcs_xyz789...
&scope=api.read+api.write
```

**(B) Token Response:** The server validates the client, issues a bearer token:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api.read api.write"
}
```

Note: The client credentials grant **does NOT return a refresh token**.
The client must re-authenticate when the access token expires. This is a
deliberate security design — refresh tokens are tied to user sessions and
are inappropriate for M2M contexts.

### When to Use

| Use Case | Why Client Credentials |
|---|---|
| **Service-to-service APIs** | Microservice A calling microservice B; no user context needed |
| **Background jobs / cron** | Nightly batch processing, data export, report generation |
| **CI/CD pipelines** | Deploy scripts accessing artifact registries or secrets vaults |
| **Scheduled token refresh** | Internal services refreshing API tokens periodically |
| **System-to-system webhooks** | Server delivering events to another server |

### When NOT to Use

| Wrong Use Case | Correct Alternative |
|---|---|
| **SPA (React/Vue/Angular)** | Authorization Code + PKCE |
| **Mobile apps** | Authorization Code + PKCE |
| **CLI tools (user-initiated)** | Authorization Code + PKCE (browser redirect) |
| **Desktop apps** | Authorization Code + PKCE |
| **User-context APIs** | Authorization Code grant (has user identity) |

The critical distinction: client credentials produce tokens with **no user
identity**. The `sub` claim is typically the client itself. Any application
that needs to act on behalf of a user must use the authorization code flow.

---

## 2. Client Secret Storage

Client secrets are the shared symmetric keys used to authenticate
confidential clients. They are high-value targets and must be protected
with the same rigor as user passwords.

### Principles

1. **Never store plaintext secrets in the database.** A database dump
   must not expose usable credentials.
2. **Use a slow, salted hash function.** Argon2id (preferred) or bcrypt.
   SHA-256 alone is insufficient — it is too fast and vulnerable to GPU
   brute force.
3. **Secret rotation should be supported** without downtime.
4. **The plaintext secret is returned only once** at creation time.

### Hashing with Argon2id (GGID Current Approach)

GGID uses Argon2id for both user passwords and client secrets via
`pkg/crypto/crypto.go`:

```go
package crypto

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
    "golang.org/x/crypto/argon2"
)

const (
    argonMemory      = 64 * 1024 // 64 MB
    argonIterations  = 3
    argonParallelism = 2
    argonKeyLength   = 32
    argonSaltLength  = 16
)

// HashClientSecret hashes a client secret using Argon2id.
func HashClientSecret(secret string) (string, error) {
    salt := make([]byte, argonSaltLength)
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return "", fmt.Errorf("salt generation: %w", err)
    }
    hash := argon2.IDKey([]byte(secret), salt, argonIterations,
        argonMemory, argonParallelism, argonKeyLength)
    return fmt.Sprintf("argon2id$%d$%d$%d$%s.%s",
        argonIterations, argonMemory, argonParallelism,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyClientSecret compares a plaintext secret against stored hash.
func VerifyClientSecret(secret, encoded string) (bool, error) {
    var iter, mem uint32
    var par uint8
    var saltB64, hashB64 string
    _, err := fmt.Sscanf(encoded, "argon2id$%d$%d$%d$%s",
        &iter, &mem, &par, &saltB64)
    if err != nil {
        return false, err
    }
    parts := splitOnDot(saltB64) // salt.hash
    salt, _ := base64.RawStdEncoding.DecodeString(parts[0])
    expectedHash, _ := base64.RawStdEncoding.DecodeString(parts[1])
    actualHash := argon2.IDKey([]byte(secret), salt, iter, mem, par, len(expectedHash))
    return constantTimeEqual(actualHash, expectedHash), nil
}
```

### Secret Prefix for Lookup Optimization

Argon2id is intentionally slow (~100ms per verification). Looking up a
client by `client_id` first, then verifying the hash is standard. But when
authenticating by **secret alone** (some legacy flows), a prefix index
enables efficient DB lookup without scanning all clients:

```go
// secretPrefix returns the first 8 hex chars of SHA-256(secret).
// Stored as an indexed column for O(1) lookup, then full Argon2id verify.
func secretPrefix(secret string) string {
    h := sha256.Sum256([]byte(secret))
    return hex.EncodeToString(h[:4]) // 8 hex chars = 4 bytes
}
```

```sql
-- Migration: add prefix index for fast secret lookup
ALTER TABLE oauth_clients ADD COLUMN client_secret_prefix VARCHAR(8);
CREATE INDEX idx_client_secret_prefix ON oauth_clients(client_secret_prefix);
```

**Security note:** The prefix reveals 32 bits of information about the
secret. This is acceptable because:
- 32 bits is insufficient to brute-force the full secret
- The full Argon2id hash still protects the actual value
- The prefix merely narrows the candidate set for verification

### Secret Rotation Policy

| Rotation Trigger | Action |
|---|---|
| **Suspected compromise** | Immediate rotation, revoke old secret |
| **Scheduled (90 days)** | Proactive rotation for high-privilege clients |
| **Team member departure** | Rotate if member had secret access |
| **Annual (low-risk clients)** | Minimum rotation cadence |

```go
// RotateClientSecret generates a new secret, invalidating the old one.
// The caller must have already verified the old secret (or be an admin).
func (s *OAuthService) RotateClientSecret(ctx context.Context,
    tenantID uuid.UUID, clientID, oldSecret string) (string, error) {
    client, err := s.clientRepo.GetClientByID(ctx, tenantID, clientID)
    if err != nil {
        return "", errors.Unauthenticated("client not found")
    }
    if client.IsConfidential() {
        ok, _ := crypto.VerifyPassword(oldSecret, client.ClientSecretHash)
        if !ok {
            return "", errors.Unauthenticated("old secret mismatch")
        }
    }
    newSecret := generateClientSecret() // "gcs_" + 32 random bytes
    hash, err := crypto.HashPassword(newSecret)
    if err != nil {
        return "", errors.Internal("hash failed", err)
    }
    client.ClientSecretHash = hash
    _, err = s.clientRepo.UpdateClient(ctx, tenantID, clientID, client)
    if err != nil {
        return "", err
    }
    return newSecret, nil
}
```

---

## 3. mTLS-Bound Clients (RFC 8705)

RFC 8705 ("Mutual TLS Client Authentication") allows a client to
authenticate using an X.509 certificate instead of a shared secret. The
authorization server validates the TLS client certificate during the
handshake, eliminating the need for client secrets entirely.

### How It Works

1. **Client Registration:** The client registers with a certificate
   thumbprint (`x5t#S256`) instead of (or in addition to) a secret.
2. **Token Request:** The client presents its certificate during the TLS
   handshake. The server extracts the thumbprint.
3. **Authentication:** The server compares the extracted thumbprint
   against the registered value.
4. **Sender-Constrained Tokens:** Issued tokens include a `cnf` claim
   binding them to the certificate, preventing token theft and replay.

### Certificate Thumbprint Computation

The `x5t#S256` thumbprint is the SHA-256 hash of the DER-encoded
certificate, base64url-encoded:

```go
package oauth

import (
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
)

// ComputeCertThumbprint calculates x5t#S256 per RFC 8705 §3.1.
func ComputeCertThumbprint(cert *x509.Certificate) string {
    der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
    // For x5t#S256, hash the full DER certificate bytes:
    h := sha256.Sum256(cert.Raw)
    return "x5t#S256:" + base64.RawURLEncoding.EncodeToString(h[:])
}

// VerifyMTLSAuth authenticates a client using their TLS certificate.
func VerifyMTLSAuth(clientCert *x509.Certificate, registeredThumbprint string) error {
    if clientCert == nil {
        return fmt.Errorf("no client certificate provided")
    }
    thumbprint := ComputeCertThumbprint(clientCert)
    if !constantTimeEqual(thumbprint, registeredThumbprint) {
        return fmt.Errorf("certificate thumbprint mismatch")
    }
    // Also verify the certificate chain (done by Go's tls.Config.VerifyPeerCertificate
    // for CA-signed certs; for self-signed, compare against stored cert directly).
    return nil
}
```

### Sender-Constrained Tokens

mTLS-bound tokens include a confirmation claim:

```json
{
  "iss": "https://auth.ggid.dev",
  "sub": "service-payment",
  "aud": "api.ggid.dev",
  "exp": 1719993600,
  "cnf": {
    "x5t#S256": "x5t#S256:abc123..."
  }
}
```

On every API call, the resource server validates:
1. The access token signature and expiry
2. The `cnf.x5t#S256` claim matches the connecting client's certificate

This makes stolen tokens useless — an attacker cannot use the token
without the private key corresponding to the bound certificate.

### Why mTLS Is More Secure Than Secret-Based

| Factor | Shared Secret | mTLS |
|---|---|---|
| **Credential type** | Symmetric (both sides know it) | Asymmetric (private key never sent) |
| **Interception risk** | Secret visible in request body/headers | Private key never transmitted |
| **Token replay** | Bearer token can be replayed | Token bound to cert; useless without key |
| **Rotation cost** | Generate hash, update DB, distribute | Generate new cert, update thumbprint |
| **Compromise impact** | Secret = full access | Key compromise requires cert revocation |

### Extracting Client Certs in Go

```go
// In the HTTP handler, extract the verified client certificate.
func extractClientCert(r *http.Request) *x509.Certificate {
    if len(r.TLS.PeerCertificates) == 0 {
        return nil
    }
    return r.TLS.PeerCertificates[0] // leaf certificate
}
```

The TLS listener must be configured with `ClientAuth: tls.RequireAndVerifyClientCert`:

```go
tlsConfig := &tls.Config{
    ClientAuth: tls.RequireAndVerifyClientCert,
    ClientCAs:  caCertPool, // for CA-signed; nil for self-signed
}
```

---

## 4. JWT Assertion Auth (RFC 7523)

RFC 7523 defines JWT profiles for OAuth 2.0, including **`private_key_jwt`** —
a client authentication method where the client signs a JWT with its private
key instead of sending a shared secret.

### How It Works

1. **Client Registration:** The client registers a public key (JWKS URI or
   inline JWKS).
2. **Token Request:** The client creates a signed JWT assertion:
   ```http
   POST /oauth/token HTTP/1.1
   grant_type=client_credentials
   &client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
   &client_assertion=eyJhbGciOiJSUzI1NiIs...
   ```
3. **Verification:** The server validates the JWT signature using the
   client's registered public key, then verifies the claims.

### JWT Assertion Structure

```json
{
  "iss": "gcid_abc123",        // MUST equal client_id
  "sub": "gcid_abc123",        // MUST equal client_id
  "aud": "https://auth.ggid.dev/oauth/token",  // token endpoint
  "jti": "a-unique-nonce-xyz", // unique per request (replay prevention)
  "exp": 1719993600,           // expiry (short-lived: 5 min max)
  "iat": 1719993300            // issued at
}
```

### Go Code: JWT Assertion Verification

```go
package oauth

import (
    "crypto/rsa"
    "errors"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// ClientAssertionClaims holds validated claims from client_assertion JWT.
type ClientAssertionClaims struct {
    ClientID string
    JTI      string
    Exp      time.Time
}

// ValidateClientAssertion validates a private_key_jwt assertion per RFC 7523 §3.
func ValidateClientAssertion(assertion string, expectedClientID string,
    clientPubKey *rsa.PublicKey, issuer string) (*ClientAssertionClaims, error) {

    if assertion == "" {
        return nil, errors.New("client_assertion is required")
    }

    // Parse and verify signature with the client's registered public key.
    claims := jwt.MapClaims{}
    _, err := jwt.ParseWithClaims(assertion, claims, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return clientPubKey, nil
    })
    if err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }

    // iss MUST equal client_id (RFC 7523 §3.1.1).
    if claims["iss"] != expectedClientID {
        return nil, errors.New("iss must match client_id")
    }

    // sub MUST equal client_id (RFC 7523 §3.1.2).
    if claims["sub"] != expectedClientID {
        return nil, errors.New("sub must match client_id")
    }

    // aud MUST be the token endpoint URL (RFC 7523 §3.1.3).
    aud, _ := claims["aud"].(string)
    if aud != issuer {
        return nil, errors.New("aud must be the token endpoint")
    }

    // exp MUST be present and in the future.
    exp, ok := claims["exp"].(float64)
    if !ok {
        return nil, errors.New("missing exp claim")
    }
    if time.Now().Unix() > int64(exp) {
        return nil, errors.New("assertion expired")
    }

    jti, _ := claims["jti"].(string)
    return &ClientAssertionClaims{
        ClientID: expectedClientID,
        JTI:      jti,
        Exp:      time.Unix(int64(exp), 0),
    }, nil
}

// Replay prevention: store jti in Redis with TTL = assertion max age.
func checkJTIReplay(jti string) error {
    if jti == "" {
        return nil // jti is optional but recommended
    }
    // SETNX jti:abc123 "1" EX 600
    key := "client_assertion_jti:" + jti
    ok, err := redis.SetNX(ctx, key, "1", 10*time.Minute).Result()
    if err != nil {
        return err
    }
    if !ok {
        return errors.New("replay detected: jti already used")
    }
    return nil
}
```

### Why JWT Assertion Is Better for Distributed Clients

| Factor | Shared Secret | private_key_jwt |
|---|---|---|
| **Secret distribution** | Secret shared to all instances | Each instance has same private key |
| **Compromise radius** | DB leak exposes all secrets | DB leak exposes public keys only |
| **Non-repudiation** | No (symmetric) | Yes (only key holder can sign) |
| **Audience binding** | None | `aud` claim limits token endpoint |
| **Replay prevention** | None | `jti` + Redis SETNX |
| **Multi-tenant** | Separate secret per client | Same key, different claims |

---

## 5. Public vs Confidential Clients

OAuth 2.0 classifies clients into two types based on their ability to
securely store credentials:

### Public Clients

Public clients **cannot** keep a secret. Any code or configuration shipped
to the end-user's device is effectively public.

| Type | Examples | Why Public |
|---|---|---|
| **SPA** | React, Vue, Angular | Source visible in browser DevTools |
| **Mobile** | iOS/Android native apps | APK/IPA can be decompiled |
| **Desktop** | Electron, native apps | Binary analysis reveals embedded secrets |
| **CLI tools** | Developer CLIs | Config files on shared machines |

### Confidential Clients

Confidential clients **can** keep a secret. They run on servers where the
secret never reaches end-user devices.

| Type | Examples | Why Confidential |
|---|---|---|
| **Server apps** | Go/Java/Node backend | Secret in env vars, never shipped |
| **Microservices** | M2M API callers | Secret in vault/KMS |
| **CI/CD** | GitHub Actions, Jenkins | Secret in CI secrets store |

### Classification Matrix

```
                    Has Secret?
                   /           \
               YES              NO
                |               |
        CONFIDENTIAL          PUBLIC
           |                     |
   - Server apps             - SPA
   - Microservices           - Mobile apps
   - CI/CD                   - CLI tools
           |                     |
   Auth methods:             Auth methods:
   - client_secret_basic     - none (public)
   - client_secret_post      - MUST use PKCE
   - client_secret_jwt
   - private_key_jwt
   - tls_client_auth
```

### Why Public Clients MUST Use PKCE

PKCE (Proof Key for Code Exchange, RFC 7636) replaces the client secret
with a dynamically generated challenge-response. Even if an attacker
intercepts the authorization code (e.g., via a malicious app registering
a custom URL scheme), they cannot exchange it without the code verifier.

```
1. App generates: code_verifier (random 43-128 char string)
2. App computes:  code_challenge = BASE64URL(SHA256(code_verifier))
3. App sends:     GET /authorize?code_challenge=xxx&code_challenge_method=S256
4. Server stores challenge with the authorization code
5. App sends:     POST /token?code=xxx&code_verifier=original_verifier
6. Server hashes verifier, compares to stored challenge
```

Without PKCE, a public client's `client_id` is the only credential. An
attacker who intercepts the code can exchange it with just the `client_id`.

---

## 6. Client Authentication Methods Comparison

OAuth 2.0 defines multiple methods for authenticating clients at the token
endpoint. The choice depends on client type, deployment environment, and
threat model.

### Method Reference

| Method | RFC | Description |
|---|---|---|
| `none` | 6749 | No authentication (public clients only) |
| `client_secret_basic` | 6749 | HTTP Basic auth header with client_id:secret |
| `client_secret_post` | 6749 | Secret in form body |
| `client_secret_jwt` | 7523 | HMAC-SHA256 JWT signed with client secret |
| `private_key_jwt` | 7523 | RSA/ECDSA JWT signed with client private key |
| `tls_client_auth` | 8705 | mTLS with CA-issued certificate |
| `self_signed_tls_client_auth` | 8705 | mTLS with self-signed certificate |

### Security Ranking (Most to Least Secure)

```
1. tls_client_auth         [Best: asymmetric, transport-bound, replay-proof]
2. private_key_jwt         [Strong: asymmetric, non-repudiation, jti replay prevention]
3. self_signed_tls_client_auth [Strong: asymmetric but cert management overhead]
4. client_secret_jwt       [Medium: HMAC but secret is shared]
5. client_secret_basic     [Baseline: secret over TLS; simplest]
6. client_secret_post      [Weak: secret in body; logged by intermediaries]
7. none                    [No auth: PKCE required]
```

### Comparison Table

| Method | Secret Type | Replay-Proof | Non-Repudiation | Key Exchange | Best For |
|---|---|---|---|---|---|
| `tls_client_auth` | Asymmetric (cert) | Yes (sender-constrained) | N/A | Cert thumbprint registration | High-security M2M |
| `private_key_jwt` | Asymmetric (RSA/ECDSA) | Yes (jti) | Yes | JWKS URI registration | Distributed clients |
| `client_secret_jwt` | Symmetric (secret) | Partial (jti) | No | Secret registration | Legacy improvement |
| `client_secret_basic` | Symmetric (secret) | No | No | Secret registration | Simple server apps |
| `client_secret_post` | Symmetric (secret) | No | No | Secret registration | Avoid; use basic |
| `none` | None | N/A | N/A | N/A | Public + PKCE |

### When Each Is Appropriate

- **`tls_client_auth`**: Zero-trust internal networks, regulated industries
  (finance, healthcare). Requires PKI infrastructure.
- **`private_key_jwt`**: Microservice meshes, multi-region deployments
  where secret distribution is a liability. Works over plaintext HTTP
  (still recommend TLS).
- **`client_secret_basic`**: The default for most confidential clients.
  Simple, widely supported, adequate when TLS is enforced.
- **`none`**: Only for public clients with PKCE enforced.

---

## 7. Client Credential Lifecycle

Managing client credentials involves registration, rotation, revocation,
and scope management throughout the client's lifetime.

### Registration

Two models exist:

1. **Manual Registration** (admin console): An administrator creates the
   client via a management API. Suitable for internal clients and trusted
   partners.

2. **Dynamic Registration** (RFC 7591): Clients self-register via
   `POST /oauth/register`. GGID implements this:

```go
// DynamicRegistrationRequest per RFC 7591
type DynamicRegistrationRequest struct {
    ClientName              string   `json:"client_name"`
    RedirectURIs           []string `json:"redirect_uris"`
    GrantTypes             []string `json:"grant_types"`
    ResponseTypes          []string `json:"response_types"`
    TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
    Scope                  string   `json:"scope"`
}
```

### Credential Rotation

```go
// RotateClientSecret: caller proves old secret, gets new plaintext.
// Old secret is immediately invalidated.
func (s *OAuthService) RotateClientSecret(ctx context.Context,
    tenantID uuid.UUID, clientID, oldSecret string) (string, error) {
    client, err := s.clientRepo.GetClientByID(ctx, tenantID, clientID)
    if err != nil {
        return "", errors.Unauthenticated("client not found")
    }
    // Verify old secret
    ok, _ := crypto.VerifyPassword(oldSecret, client.ClientSecretHash)
    if !ok {
        return "", errors.Unauthenticated("invalid old secret")
    }
    // Generate and store new
    newSecret := generateClientSecret()
    hash, _ := crypto.HashPassword(newSecret)
    client.ClientSecretHash = hash
    s.clientRepo.UpdateClient(ctx, tenantID, clientID, client)
    return newSecret, nil
}
```

### Revocation

When a client is compromised or decommissioned, all its tokens must be
revoked immediately:

```go
// RevokeClient disables the client and revokes all tokens.
func (s *OAuthService) RevokeClient(ctx context.Context,
    tenantID uuid.UUID, clientID string) error {
    // 1. Mark client as disabled (blocks new token requests)
    client, _ := s.clientRepo.GetClientByID(ctx, tenantID, clientID)
    client.Enabled = false
    s.clientRepo.UpdateClient(ctx, tenantID, clientID, client)
    // 2. Revoke all active refresh tokens
    s.tokenRepo.RevokeAllRefreshTokens(ctx, tenantID, clientUUID)
    // 3. Active access tokens expire naturally (short-lived)
    return nil
}
```

### Scope Management

Each client should have a bounded scope set. The client credentials grant
must **never** issue scopes broader than the client's registered scope list.

```go
// requestedScopes must be a subset of client.Scopes
func validateRequestedScopes(requested, allowed []string) error {
    allowedSet := make(map[string]bool)
    for _, s := range allowed {
        allowedSet[s] = true
    }
    for _, s := range requested {
        if !allowedSet[s] {
            return fmt.Errorf("scope '%s' not allowed for this client", s)
        }
    }
    return nil
}
```

### Rate Limiting Per Client

```go
// Per-client rate limit: 100 token requests per minute.
func rateLimitedClient(clientID string) bool {
    key := "rl:client:" + clientID
    count, _ := redis.Incr(ctx, key).Result()
    if count == 1 {
        redis.Expire(ctx, key, time.Minute)
    }
    return count > 100
}
```

---

## 8. GGID Client Credentials Audit

This section audits the actual GGID OAuth service implementation against
the security best practices described above.

### What GGID Currently Has

| Feature | Status | Location |
|---|---|---|
| **Client credentials grant** | Implemented | `oauth_service.go:ClientCredentials()` |
| **Secret hashing (Argon2id)** | Implemented | `pkg/crypto/crypto.go:HashPassword()` |
| **Secret verification** | Implemented | `oauth_service.go:839` via `crypto.VerifyPassword()` |
| **Secret rotation** | Implemented | `oauth_service.go:RotateClientSecret()` |
| **Dynamic registration (RFC 7591)** | Implemented | `oauth_service.go:DynamicClientRegister()` |
| **PKCE enforcement for public clients** | Implemented | `server.go:184`, `domain.OAuthClient.RequiresPKCE()` |
| **mTLS thumbprint extraction** | Implemented | `jar_mtls.go:ExtractCertThumbprint()` |
| **mTLS sender-constrained token validation** | Implemented | `jar_mtls.go:ValidateMTLSClientAuth()` |
| **mTLS binding check** | Implemented | `jar_mtls.go:ValidateMTLSBinding()` |
| **RFC 7523 JWT assertion validation** | Implemented | `rfc7523.go:ValidateClientAssertion()` |
| **RFC 7523 JWT client auth entry point** | Implemented | `rfc7523.go:ValidateJWTClientAuth()` |
| **Public/Confidential classification** | Implemented | `domain/models.go:ClientType` |
| **Client enable/disable** | Implemented | `domain.OAuthClient.Enabled` |
| **Discovery: token_endpoint_auth_methods** | Implemented | Lists 5 methods including `tls_client_auth` |
| **Token revocation (RFC 7009)** | Implemented | `server.go:/oauth/revoke` |
| **Introspection client auth** | Implemented | `server.go:553` requires client_id + client_secret |
| **Password pepper** | Implemented | `pkg/crypto/crypto.go:SetPepper()` |

### Audit Details

**Secret Storage (GOOD):** Secrets are hashed with Argon2id via
`crypto.HashPassword()` — the same function used for user passwords. The
domain model field is named `ClientSecretHash`, confirming plaintext is
never stored. The `createClient()` function generates the plaintext
secret, hashes it, stores the hash, and returns the plaintext only once.

**Client Credentials Grant (GOOD with caveats):** The `ClientCredentials()`
method:
1. Looks up the client by ID
2. Verifies the secret for confidential clients
3. Checks the client is enabled
4. Validates the client supports `client_credentials` grant
5. Issues an access token

**Gap:** The token endpoint (`server.go:293`) reads `client_id` and
`client_secret` only from form body (`r.FormValue()`). It does NOT support
HTTP Basic auth (`r.BasicAuth()`) for `client_secret_basic` — even though
discovery advertises it. Clients sending Basic auth would fail.

**mTLS (PARTIAL):** GGID has `ExtractCertThumbprint()` and
`ValidateMTLSClientAuth()` for sender-constrained token verification.
However:
- The token endpoint handler does not extract or verify client certificates
- The `ClientCredentials()` grant does not check for `tls_client_auth` method
- No `tls.Config` is configured for `RequireAndVerifyClientCert`
- Client registration does not store certificate thumbprints

**JWT Assertion (PARTIAL):** `ValidateClientAssertion()` parses the JWT
and validates `iss`, `sub`, `aud`, `exp` claims. However:
- It uses `ParseUnverified()` — **signature verification is not performed**
- The comment says "production would verify against client's registered
  public key" — this is a placeholder
- No JWKS URI registration for client public keys
- No `jti` replay prevention (Redis SETNX)
- The token endpoint handler does not route `client_assertion` to this method

### What GGID Is Missing

| Gap | Severity | Impact |
|---|---|---|
| `client_secret_basic` not parsed in token endpoint | Medium | Standards non-compliance; clients using Basic auth fail |
| JWT assertion signature not verified | High | Any client can forge an assertion with arbitrary claims |
| mTLS not wired into token endpoint | Medium | `tls_client_auth` advertised but not functional |
| No `jti` replay prevention for assertions | Medium | Assertion capture/replay within expiry window |
| No per-client rate limiting on token endpoint | Medium | Brute-force and DoS vulnerability |
| Client secret has no expiry (`ClientSecretExpiresAt`) | Low | No enforced rotation schedule |
| No scope intersection validation in `ClientCredentials()` | Medium | Client could request scopes beyond registration |
| No `x5t#S256` claim (`cnf`) in issued access tokens | Medium | Tokens are bearer, not sender-constrained |

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

#### P0: Fix JWT Assertion Signature Verification (CRITICAL)

**Current:** `ValidateClientAssertion()` uses `jwt.ParseUnverified()`.
**Risk:** Any attacker who knows a `client_id` can forge an assertion.
**Fix:** Parse with the client's registered public key. Require JWKS URI
or inline JWKS during registration. Fall back to `ParseUnverified` only
for claim extraction, then verify signature separately.

**Effort:** 4-6 hours (JWKS storage, key resolution, signature validation)

```go
// Required fix: verify signature, not just claims
keyFunc := func(t *jwt.Token) (interface{}, error) {
    kid, _ := t.Header["kid"].(string)
    pubKey, err := s.resolveClientKey(clientID, kid)
    if err != nil {
        return nil, fmt.Errorf("key resolution: %w", err)
    }
    return pubKey, nil
}
_, err = jwt.Parse(assertion, keyFunc) // Actually verifies signature
```

#### P1: Add `client_secret_basic` Support to Token Endpoint

**Current:** Token endpoint only reads `r.FormValue("client_secret")`.
**Risk:** Discovery advertises `client_secret_basic` but it does not work.
**Fix:** Check `r.BasicAuth()` first, fall back to form body.

**Effort:** 1 hour

```go
clientID, clientSecret, ok := r.BasicAuth()
if !ok {
    clientID = r.FormValue("client_id")
    clientSecret = r.FormValue("client_secret")
}
```

#### P2: Wire mTLS Client Authentication into Token Endpoint

**Current:** `ExtractCertThumbprint()` and `ValidateMTLSClientAuth()`
exist but are never called during client credentials grant.
**Fix:** Extract client cert from `r.TLS.PeerCertificates`, compute
thumbprint, verify against registered `tls_client_auth` thumbprint.

**Effort:** 3-4 hours (TLS config, cert extraction, thumbprint comparison,
client registration update)

#### P3: Add Per-Client Rate Limiting and Scope Validation

**Current:** No rate limiting or scope intersection check in
`ClientCredentials()`.
**Fix:** Add Redis-backed per-client rate limit. Validate requested scopes
are a subset of registered scopes.

**Effort:** 2-3 hours

#### P4: Implement `jti` Replay Prevention for JWT Assertions

**Current:** `jti` is extracted but not checked against Redis.
**Fix:** Store `jti` in Redis with TTL = assertion max lifetime. Reject
duplicate JTIs.

**Effort:** 1-2 hours

### Summary Table

| Item | Severity | Effort | RFC |
|---|---|---|---|
| P0: JWT assertion signature verification | Critical | 4-6h | RFC 7523 |
| P1: `client_secret_basic` in token endpoint | Medium | 1h | RFC 6749 §2.3.1 |
| P2: mTLS wired into token endpoint | Medium | 3-4h | RFC 8705 |
| P3: Rate limiting + scope validation | Medium | 2-3h | RFC 6749 |
| P4: `jti` replay prevention | Medium | 1-2h | RFC 7523 |

**Total estimated effort:** 11-16 hours for full compliance.

### Architectural Recommendation

GGID has built strong foundational primitives (Argon2id hashing, mTLS
thumbprint utilities, RFC 7523 claim validation, PKCE enforcement). The
primary gap is **wiring**: the security-critical validation functions
exist but are not connected to the request pipeline. The highest-ROI
work is connecting these existing components to the token endpoint
handler, not building new ones.

---

## References

- RFC 6749: The OAuth 2.0 Authorization Framework (§4.4 Client Credentials Grant)
- RFC 7591: OAuth 2.0 Dynamic Client Registration Protocol
- RFC 7523: JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication
  and Authorization Grants
- RFC 8705: Mutual-TLS Client and Certificate-Bound Access Tokens
- RFC 7636: Proof Key for Code Exchange by OAuth Public Clients (PKCE)
- RFC 9101: JWT-Secured Authorization Request (JAR)
- OWASP: OAuth 2.0 Security Best Current Practice
- NIST SP 800-63B: Digital Identity Guidelines (authentication assurance levels)

---

*Research conducted against GGID commit baseline. See
`services/oauth/internal/` for implementation details.*
