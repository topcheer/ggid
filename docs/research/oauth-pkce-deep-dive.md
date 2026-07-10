# OAuth PKCE Deep Dive — RFC 7636, OAuth 2.1, and GGID Implementation

> Research document covering the authorization code interception attack, the PKCE
> mitigation, why OAuth 2.1 makes PKCE mandatory, and a concrete analysis of
> GGID's current implementation with a hardening roadmap.

---

## 1. Overview

**PKCE** (Proof Key for Code Exchange, pronounced "pixy") is defined in
[RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636). It extends the OAuth
2.0 authorization code grant with a cryptographic binding between the client and
the authorization code.

PKCE was originally designed for **public clients** — native and mobile apps,
single-page apps, and any client that cannot keep a `client_secret` confidential.
Without a secret, the authorization code alone was enough to obtain tokens,
making interception a real threat.

**OAuth 2.1** (the in-progress consolidation of RFC 6749, 7636, 9700, and
related specs) makes PKCE **mandatory for all clients**, including confidential
ones that already present a secret. The `S256` method is required; the `plain`
method is removed entirely.

**The problem PKCE solves:** the authorization code interception attack. In the
classic flow, the authorization code travels through the browser (via a redirect
to the client's redirect URI). An attacker who observes or intercepts that code
can redeem it for tokens before the legitimate client does.

**How PKCE fixes it:** the client proves possession of a secret
(`code_verifier`) that was *never sent through the browser*. The authorization
server binds the code to a derived `code_challenge` at issuance and verifies the
verifier at redemption. An interceptor gets the code but cannot derive the
verifier, so the code is useless to them.

---

## 2. The Interception Attack

Without PKCE, the authorization code grant is vulnerable to interception in any
environment where the redirect callback is observable:

```
Step 1: Native app opens the system browser → https://as/authorize?...
Step 2: User authenticates and consents at the AS
Step 3: AS redirects → myapp://callback?code=ABC123
Step 4: Attacker intercepts the redirect (custom-scheme hijack, log exposure,
        malicious app registering the same scheme, reverse proxy logging)
Step 5: Attacker POSTs to /token with code=ABC123 → receives access_token
```

**Why this works without PKCE:** for a public client (no secret), the token
endpoint accepts the code + `client_id` + `redirect_uri` and issues tokens. The
code is a bearer credential. Anyone who holds it within its short lifetime can
redeem it.

Interception vectors in practice:

| Vector | Environment | Likelihood |
|---|---|---|
| Custom URI scheme hijacking | iOS / Android | High (two apps can register) |
| OS-level intent interception | Android | Medium |
| Reverse proxy / WAF logging | Server-side SPA | Medium |
| Referer header leakage | Browser → 3rd party | Low–Medium |
| Browser history / extension access | Web app | Low |

**With PKCE:** the attacker must also provide the `code_verifier`, which only
exists in the client's memory and was never transmitted over the network or
through the browser. The challenge is a one-way hash — the verifier cannot be
recovered from it. The interception is neutralized.

---

## 3. PKCE Flow

### Registration (Authorization Request)

```
Client                                          Authorization Server
  |                                                    |
  |-- generate code_verifier (43-128 chars)           |
  |-- compute code_challenge = B64URL(SHA256(verifier))|
  |-- GET /authorize?                                  |
  |     response_type=code                            |
  |     &code_challenge=XYZ...                         |
  |     &code_challenge_method=S256                   |
  |------------------------------------------------- -->|
  |                                                    | store challenge
  |<-------- 302 redirect ?code=AUTHCODE --------------| with code
```

### Token Exchange

```
Client                                          Authorization Server
  |                                                    |
  |-- POST /token                                      |
  |     grant_type=authorization_code                 |
  |     &code=AUTHCODE                                 |
  |     &code_verifier=<the secret>                    |
  |------------------------------------------------- -->|
  |                                                    | recompute:
  |                                                    | B64URL(SHA256(verifier))
  |                                                    | == stored challenge?
  |                                                    | YES → issue tokens
  |                                                    | NO  → reject (interception)
  |<--------------- { access_token, ... } -------------|
```

### S256 vs plain

| Method | `code_challenge` | Security | OAuth 2.1 |
|---|---|---|---|
| **S256** | `BASE64URL(SHA256(ASCII(verifier)))` | Verifier unrecoverable; challenge safe to expose | **Mandatory** |
| **plain** | `code_verifier` itself | Zero added security if challenge is intercepted | **Removed** |

`plain` exists only for clients running on platforms that cannot compute SHA-256
(legacy embedded systems). It provides **no benefit** over no PKCE if the
authorization request — and thus the challenge — is observed by the attacker.
Modern OAuth 2.1 eliminates it.

---

## 4. Why OAuth 2.1 Makes PKCE Mandatory

RFC 7636 scoped PKCE to public clients. The security community later recognized
that **confidential clients benefit too**, because code interception is not
limited to public-client redirect mechanisms:

1. **Defense in depth.** Even a confidential client's redirect can be logged,
   leaked, or intercepted. Client authentication at the token endpoint is a
   second factor, but PKCE provides a third, independent binding.
2. **[RFC 9700](https://datatracker.ietf.org/doc/html/rfc9700)** (OAuth 2.0
   Security Best Current Practice) recommends PKCE for **all** authorization
   code flows, regardless of client type.
3. **OAuth 2.1 draft** makes `S256` mandatory and removes `plain`. Every client
   must send `code_challenge` + `code_challenge_method=S256`.
4. **Industry trend.** Google, GitHub, and Microsoft already require PKCE for
   new OAuth registrations; Apple Sign-In mandates it.

The takeaway: PKCE is no longer a mobile-only feature. It is baseline
authorization-code security for every client type.

---

## 5. Code Verifier Generation

RFC 7636 §4.1 constrains the verifier:

- **Length:** 43–128 characters
- **Character set:** `[A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"`
  (unreserved URI characters from RFC 3986)
- **Entropy:** minimum 256 bits → 32 random bytes → 43 base64url characters

**Common mistake:** using `Math.random()` (JavaScript) or a non-CSPRNG. The
verifier must come from a cryptographically secure random source.

**Go reference implementation:**

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
)

// generateCodeVerifier produces a cryptographically secure PKCE code_verifier.
// 32 random bytes → 43 base64url chars = 256 bits of entropy (RFC 7636 minimum).
func generateCodeVerifier() (verifier, challenge string, err error) {
    b := make([]byte, 32)
    if _, err = rand.Read(b); err != nil {
        return "", "", err
    }
    verifier = base64.RawURLEncoding.EncodeToString(b) // no padding
    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])
    return verifier, challenge, nil
}
```

GGID's `pkg/crypto.GenerateRandomToken(32)` already uses `crypto/rand` and is
used for authorization codes themselves; the same primitive can back verifier
generation in the SDKs.

---

## 6. GGID Implementation Analysis

GGID implements PKCE across three layers:

- **Domain** (`services/oauth/internal/domain/models.go`): `AuthorizationCode`
  carries `CodeChallenge` / `CodeChallengeMethod`; `ValidatePKCE()` does the
  comparison. `OAuthClient.RequiresPKCE()` = `RequirePKCE || IsPublic()`.
- **Service** (`oauth_service.go`): `CreateAuthorizationCode` enforces PKCE for
  public/flagged clients and defaults method to `S256`.
  `ExchangeAuthorizationCode` calls `code.ValidatePKCE(req.CodeVerifier)`.
- **Server** (`server.go`) + **Config** (`conf.go`): a global `RequirePKCE`
  toggle validates `code_challenge` presence at the authorize endpoint.

### Compliance table

| Requirement | Current state | Compliant? | Action needed |
|---|---|---|---|
| `code_verifier` validation on token endpoint | `ValidatePKCE()` called at exchange (oauth_service.go:274) | Yes | — |
| `code_challenge` stored with auth code | Persisted via `pg_repo.go` columns | Yes | — |
| S256 method support | Full: SHA-256 + base64url in `ValidatePKCE` | Yes | — |
| plain method support | Allowed (models.go:130, discovery advertises it) | **No** — OAuth 2.1 removes it | Reject `plain`; advertise S256 only |
| `RequirePKCE` global default | `Default()` does **not** set it → `false` | **No** | Set `true` for production |
| Per-client PKCE enforcement | `RequirePKCE` flag on `OAuthClient` | Yes (opt-in) | Default new confidential clients to `true` |
| Constant-time comparison | `ValidatePKCE` uses `==` (models.go:131,135) | **No** — timing side-channel | Use `subtle.ConstantTimeCompare` |
| Downgrade protection | No check preventing S256→plain fallback | **No** | Reject if stored method differs from presented |

### Key findings

```go
// models.go — current ValidatePKCE (simplified)
func (c *AuthorizationCode) ValidatePKCE(verifier string) bool {
    if c.CodeChallenge == "" { return true }       // PKCE optional for this code
    if verifier == "" { return false }
    switch c.CodeChallengeMethod {
    case "plain", "":  return verifier == c.CodeChallenge          // timing-vulnerable
    case "S256":       // recompute hash, == compare               // timing-vulnerable
    }
}
```

Three gaps:

1. **`plain` still accepted.** Discovery advertises `["S256","plain"]`. OAuth 2.1
   removes `plain`; an attacker can't exploit it directly here, but retaining it
   signals weaker posture and prevents strict-mode clients from trusting the AS.
2. **`RequirePKCE` defaults to `false`.** Confidential clients without the
   per-client flag get no PKCE enforcement — exactly the gap OAuth 2.1 closes.
3. **Non-constant-time comparison.** `verifier == c.CodeChallenge` and
   `encoded == c.CodeChallenge` leak length/timing information via the `==`
   operator's early-exit behavior. Realistically hard to exploit over a network,
   but trivially fixable and expected in security-critical code.

---

## 7. Security Analysis

| Threat | Mitigated by PKCE? | Notes |
|---|---|---|
| Code replay | Yes (code consumed atomically) | `ConsumeCode` marks used; second redemption fails |
| Timing attack on challenge | Partially | Current `==` is not constant-time — fix with `subtle.ConstantTimeCompare` |
| Downgrade (S256 → plain) | No | GGID accepts both; must reject `plain` and enforce stored method |
| Malicious auth request injection | No | Use `state` parameter (CSRF) + exact `redirect_uri` match |
| Token theft after issuance | No | Use sender-constrained tokens: DPoP or mTLS |

**PKCE vs `state`:** complementary, not redundant. PKCE binds the code to the
client's verifier (anti-interception). `state` binds the callback to the
original request (anti-CSRF). Both are required for a complete defense.

**Strongest combination — PKCE + DPoP:** PKCE binds the *code* to the client;
[DPoP](https://datatracker.ietf.org/doc/html/rfc9449) (Demonstrating Proof of
Possession) binds the issued *access token* to the client's private key. Together
they cover both legs of the flow: code interception is neutralized by PKCE, and
token replay/theft is neutralized by DPoP. This is the OAuth 2.1 + Web3-era
gold standard.

---

## 8. SDK Impact

All GGID SDKs (Go, Node, Java) should generate and send PKCE **by default**, with
no developer opt-in required. The `code_verifier` lives only in the SDK's
memory; the `code_challenge` (S256) is sent in the authorize request.

**Go SDK:**

```go
verifier, challenge, _ := generateCodeVerifier() // see §5
authURL := authorizeURL + "?code_challenge=" + challenge +
    "&code_challenge_method=S256&..." // redirect user
// on callback:
token, _ := client.ExchangeCode(ctx, code, verifier)
```

**Node SDK** (WebCrypto — works in browser and server):

```js
const verifier = base64url(crypto.getRandomValues(new Uint8Array(32)));
const digest   = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
const challenge = base64url(new Uint8Array(digest));
```

**Java SDK** (`java.security.MessageDigest`):

```java
byte[] verifierBytes = new byte[32];
SecureRandom.getInstanceStrong().nextBytes(verifierBytes);
String verifier  = Base64.getUrlEncoder().withoutPadding().encodeToString(verifierBytes);
byte[] hash      = MessageDigest.getInstance("SHA-256").digest(verifier.getBytes(StandardCharsets.US_ASCII));
String challenge = Base64.getUrlEncoder().withoutPadding().encodeToString(hash);
```

SDK default behavior: always send `code_challenge_method=S256`, always store the
verifier in session/memory, always include `code_verifier` at token exchange.

---

## 9. Roadmap

| Phase | Task | Effort |
|---|---|---|
| 1 | Audit: confirm `ValidatePKCE` covers all token-exchange paths | 0.5 day |
| 2 | Reject `plain`; advertise only `S256` in discovery | 0.5 day |
| 3 | Switch `ValidatePKCE` to `subtle.ConstantTimeCompare` | 0.5 day |
| 4 | Set `RequirePKCE=true` as production default; default new confidential clients to `RequirePKCE` | 0.5 day |
| 5 | Update Go/Node/Java SDKs to send PKCE by default | 1 day |

**Total estimate:** ~2–3 days. After phase 4, GGID is OAuth 2.1-ready for PKCE.

---

*References: RFC 7636 (PKCE), RFC 9700 (Security BCP), OAuth 2.1 draft, RFC 9449 (DPoP).*
