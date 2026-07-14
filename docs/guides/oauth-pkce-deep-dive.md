# OAuth PKCE Deep Dive

code_verifier/code_challenge generation, S256 vs plain, per-client enforcement, migration, testing, and mobile/SPA best practices.

## Overview

PKCE (Proof Key for Code Exchange, RFC 7636) prevents authorization code interception attacks. It binds the authorization code to a verifier known only to the client.

## Flow

```
1. Client generates code_verifier (random string)
2. Client computes code_challenge = SHA256(code_verifier)
3. Client sends code_challenge in authorization request
4. Authorization server stores code_challenge with the code
5. Client receives authorization code
6. Client sends code + code_verifier to token endpoint
7. Server verifies SHA256(code_verifier) == stored code_challenge
8. If match → issue tokens; if mismatch → reject
```

If an attacker intercepts the code, they can't exchange it without the code_verifier.

## code_verifier Generation

```javascript
// Generate 43-128 character random string
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64urlEncode(array);
  // Result: 43 chars, high entropy
}
```

### Requirements

| Rule | Value |
|------|-------|
| Length | 43-128 characters |
| Charset | Unreserved URI chars `[A-Z][a-z][0-9]-._~` |
| Entropy | ≥256 bits (32 random bytes → 43 base64url chars) |
| Per session | New verifier per authorization request |
| Single use | Never reuse across requests |

## code_challenge Generation

### S256 (Required)

```javascript
function generateCodeChallenge(verifier) {
  const data = new TextEncoder().encode(verifier);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return base64urlEncode(new Uint8Array(hash));
}
```

### Plain (Discouraged)

```
code_challenge = code_verifier  // No hashing
```

| Method | Security | When to Use |
|--------|----------|-------------|
| S256 | High (SHA-256) | Always (default) |
| plain | Low (no protection) | Only if client can't compute SHA-256 (rare) |

**GGID rejects `plain` by default.** Only S256 is accepted.

## Authorization Request

```bash
GET /api/v1/oauth/authorize?
  response_type=code
  &client_id=client-123
  &redirect_uri=https://app.example.com/callback
  &scope=openid+profile
  &state=random-state
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
```

## Token Exchange

```bash
POST /api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=AUTH_CODE
&redirect_uri=https://app.example.com/callback
&client_id=client-123
&code_verifier=dBjftJeZ4CVKjm7gCL0m4QyO5OqO5oWvAB4O9Gr9l2Y
```

### Verification

```go
func verifyPKCE(storedChallenge, verifier, method string) error {
    switch method {
    case "S256":
        h := sha256.Sum256([]byte(verifier))
        computed := base64.RawURLEncoding.EncodeToString(h[:])
        if computed != storedChallenge {
            return ErrPKCEVerificationFailed
        }
    case "plain":
        if verifier != storedChallenge {
            return ErrPKCEVerificationFailed
        }
    default:
        return ErrUnknownMethod
    }
    return nil
}
```

## Per-Client Enforcement

```bash
# Require PKCE for specific clients
PATCH /api/v1/oauth/clients/{client_id}
{"require_pkce": true}
# → Authorization requests without code_challenge are rejected
```

### Enforcement Matrix

| Client Type | PKCE Required | Client Secret |
|------------|--------------|---------------|
| SPA (browser) | ✅ Required | Not used |
| Mobile app | ✅ Required | Not used |
| Web server app | ✅ Recommended | Optional |
| Backend service | N/A | Required (client_credentials) |

## Migration from Non-PKCE

### Phase 1: Support (Current)

```bash
# Client without PKCE still works (backward compat)
GET /authorize?response_type=code&client_id=old-client&...
# → No code_challenge → No PKCE verification at token exchange
```

### Phase 2: Warn

```bash
# Console warning for non-PKCE clients
# "This client is not using PKCE. We recommend enabling PKCE."
```

### Phase 3: Enforce for New Clients

```bash
# New client registrations require PKCE
POST /api/v1/oauth/register
{"client_name": "New App", "require_pkce": true}
```

### Phase 4: Deprecate Non-PKCE

```bash
# Existing clients must migrate
# Non-PKCE requests return warning header
Deprecation: PKCE recommended
```

## Testing

### Unit Test

```go
func TestPKCEVerification(t *testing.T) {
    verifier := "dBjftJeZ4CVKjm7gCL0m4QyO5OqO5oWvAB4O9Gr9l2Y"
    challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

    // Correct verifier
    assert.NoError(t, verifyPKCE(challenge, verifier, "S256"))

    // Wrong verifier
    assert.Error(t, verifyPKCE(challenge, "wrong", "S256"))

    // Plain method rejected
    assert.Error(t, verifyPKCE(verifier, verifier, "plain"))
}
```

### E2E Test

```javascript
test('full PKCE flow', async () => {
  const verifier = generateCodeVerifier();
  const challenge = await generateCodeChallenge(verifier);

  // Authorize
  const code = await authorize({ code_challenge: challenge, code_challenge_method: 'S256' });

  // Exchange with correct verifier
  const tokens = await exchangeCode(code, verifier);
  assert(tokens.access_token);

  // Exchange with wrong verifier should fail
  await expect(exchangeCode(code, 'wrong')).rejects.toThrow();
});
```

## Best Practices

### SPA

```javascript
// Use Web Crypto API (not JS crypto libraries)
const verifier = base64url(crypto.getRandomValues(new Uint8Array(32)));
const challenge = base64url(await crypto.subtle.digest('SHA-256', verifier));

// Store verifier in sessionStorage (not localStorage — cleared on tab close)
sessionStorage.setItem('pkce_verifier', verifier);
```

### Mobile

```swift
// iOS — use CryptoKit
import CryptoKit

let verifier = generateRandomString(length: 43)
let challenge = Data(SHA256.hash(data: verifier.data(using: .utf8)!))
    .base64EncodedString()
    .replacingOccurrences(of: "+", with: "-")
    .replacingOccurrences(of: "/", with: "_")
    .replacingOccurrences(of: "=", with: "")
```

### Security Notes

- Generate a new verifier for every authorization request
- Never send the verifier in the authorization request (only the challenge)
- Never log the verifier
- Use S256 always; reject plain method
- Verifier should be tied to the same session/state

## Monitoring

| Metric | Alert |
|--------|-------|
| PKCE verification failures | >1% → misconfigured clients |
| Non-PKCE authorization requests | Track for migration |
| plain method attempts | Any → log warning |

## See Also

- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth Dynamic Client Registration](oauth-dynamic-client-registration.md)
- [Token Binding Comparison](token-binding-comparison.md)
