# OAuth State Parameter Management

CSRF protection, session binding, PKCE relationship, per-flow state encoding, validation rules, and error handling.

## Overview

The `state` parameter in OAuth 2.0 prevents CSRF attacks on the authorization code flow. It binds the authorization response to the original request, ensuring the user receives tokens only for requests they initiated.

## CSRF Protection

### Attack Without State

```
1. Attacker initiates OAuth flow with client, gets authorization code
2. Attacker sends code to victim via callback URL
3. Victim's browser completes token exchange
4. Victim is now logged in as attacker (account switching attack)
```

### Defense With State

```
1. Client generates random state before redirect
2. Client stores state (cookie/sessionStorage)
3. User redirected with state to authorization server
4. Authorization server returns state in callback
5. Client verifies returned state matches stored state
6. If mismatch → reject (CSRF detected)
```

## State Generation

```javascript
// Generate cryptographically random state
function generateState() {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return base64url(array);
}
```

### Requirements

| Rule | Value |
|------|-------|
| Length | ≥128 bits entropy (16 random bytes) |
| Charset | base64url |
| Per request | New state for each authorization |
| Single use | Invalidate after callback |

## Binding to Session

### Cookie-Based (Recommended)

```javascript
// Before redirect
const state = generateState();
document.cookie = `oauth_state=${state}; Secure; HttpOnly; SameSite=Lax; Max-Age=600`;
window.location.href = buildAuthorizeURL({ state });
```

### Session-Based

```javascript
// Server-side session
req.session.oauthState = state;
res.redirect(authorizeURL);
```

### Validation on Callback

```javascript
app.get('/callback', (req, res) => {
  const returnedState = req.query.state;
  const storedState = req.cookies.oauth_state;
  
  if (!returnedState || !storedState) {
    return res.status(400).send('Missing state parameter');
  }
  if (returnedState !== storedState) {
    return res.status(403).send('State mismatch — possible CSRF');
  }
  
  // Clear state cookie
  res.clearCookie('oauth_state');
  
  // Proceed with code exchange
  exchangeCode(req.query.code);
});
```

## State Encoding (Optional Payload)

State can carry context without a server-side session:

```javascript
// Encode return-to path in state
const payload = { redirect: '/dashboard', nonce: cryptoRandom() };
const state = base64url(JSON.stringify(payload));
// → redirect: https://auth.ggid.dev/authorize?...&state=eyJyZWRpcmVjdCI6...

// On callback, decode
const payload = JSON.parse(base64urlDecode(returnedState));
```

### Encoding Rules

| Rule | Enforcement |
|------|-------------|
| Max length | 500 chars (browser URL limits) |
| Must contain nonce | Even with encoded payload |
| Never put secrets | State is visible in URL/logs |
| Sign if sensitive | HMAC with server secret |

### Signed State

```javascript
const payload = { redirect: '/admin', timestamp: Date.now() };
const data = JSON.stringify(payload);
const sig = HMAC(secret, data);
const state = base64url(data) + '.' + base64url(sig);

// Verification
const [data, sig] = state.split('.');
if (!verifyHMAC(secret, data, sig)) { reject(); }
```

## PKCE Relationship

| Parameter | Protects Against | Stored Where |
|-----------|-----------------|-------------|
| `state` | CSRF (authorization response injection) | Client cookie/session |
| `code_verifier` | Code interception (token theft) | Client memory |
| `nonce` | Token replay (ID token replay) | Client session |

All three work together:

```
Redirect:
  &state=RANDOM_STATE        (CSRF protection)
  &code_challenge=HASH       (PKCE: code binding)
  &code_challenge_method=S256
  &nonce=RANDOM_NONCE        (ID token replay protection)

Callback:
  Verify state → (CSRF check)
  Exchange code with code_verifier → (code binding)
  Verify nonce in ID token → (replay check)
```

## Validation Rules

```go
func ValidateState(returned, stored string) error {
    if returned == "" { return ErrMissingState }
    if stored == "" { return ErrNoStoredState }
    if returned != stored { return ErrStateMismatch }
    
    // Check TTL (state should be used within 10 minutes)
    if time.Since(stateCreatedAt) > 10*time.Minute {
        return ErrStateExpired
    }
    
    return nil
}
```

### Constant-Time Comparison

```go
func ValidateState(returned, stored string) error {
    if !hmac.Equal([]byte(returned), []byte(stored)) {
        return ErrStateMismatch
    }
    return nil
}
```

Use constant-time comparison to prevent timing attacks.

## Error Handling

| Error | Cause | User Action |
|-------|-------|-------------|
| Missing state | Auth server didn't return it | Retry auth flow |
| State mismatch | Different session or CSRF | Restart auth flow |
| State expired | >10 min since redirect | Restart auth flow |
| State already used | Replay attempt | Reject + log |

### User Experience

```
State mismatch → "Your session expired. Please try again."
                → Redirect to login page (fresh state)
```

**Never** tell the user "CSRF detected" or "security error" — keep it user-friendly.

## Per-Flow State

Different flows need different state handling:

| Flow | State Source | Storage |
|------|-------------|---------|
| Authorization Code | Client-generated | Cookie |
| Implicit (deprecated) | Client-generated | Cookie |
| Device Flow | Server-generated | Server session |
| SAML SSO | `RelayState` (equivalent) | Cookie |

## Monitoring

| Metric | Alert |
|--------|-------|
| State mismatch rate | >0.5% → possible attack or session config issue |
| Missing state | Any spike → client misconfigured |
| State expiry | High rate → user taking too long |

## See Also

- [OAuth PKCE Deep Dive](oauth-pkce-deep-dive.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [Session Security](session-security.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
