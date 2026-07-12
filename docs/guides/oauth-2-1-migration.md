# OAuth 2.1 Migration Guide

This guide helps you migrate from OAuth 2.0 to OAuth 2.1 in GGID — PKCE enforcement, grant deprecations, redirect URI changes, DPoP/JAR/PAR adoption.

> **Related**: [OAuth 2.1 Changes](../research/oauth-2.1-changes.md), [OAuth API](../api/oauth-api.md)

## Key Changes Summary

| Change | OAuth 2.0 | OAuth 2.1 | Impact |
|--------|-----------|-----------|--------|
| PKCE | Recommended | **Mandatory** | All clients must use PKCE |
| Implicit grant | Supported | **Removed** | SPAs must use code+PKCE |
| ROPC grant | Supported | **Removed** | Use code+PKCE or device flow |
| Redirect URI matching | Prefix allowed | **Exact only** | Update client registrations |
| Refresh rotation | Optional | **Required (public)** | Single-use refresh tokens |
| `iss` parameter | Optional | **Mandatory** | Verify in auth response |

## Step 1: Enable PKCE for All Clients

### Check Current Clients

```bash
curl https://api.ggid.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Update Client Configuration

```bash
curl -X PUT https://api.ggid.example.com/api/v1/oauth/clients/$CLIENT_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "require_pkce": true,
    "pkce_methods": ["S256"],
    "grants": ["authorization_code", "refresh_token"]
  }'
```

### Update Application Code

```javascript
// BEFORE: No PKCE
const authUrl = `${issuer}/authorize?response_type=code&client_id=${id}&redirect_uri=${uri}&state=${state}`;

// AFTER: With PKCE
const verifier = generateCodeVerifier(); // 43-128 random chars
const challenge = base64url(sha256(verifier));
const authUrl = `${issuer}/authorize?response_type=code&client_id=${id}
  &redirect_uri=${uri}&state=${state}
  &code_challenge=${challenge}&code_challenge_method=S256`;

// Token exchange includes verifier
const tokenResp = await fetch(`${issuer}/token`, {
  method: 'POST',
  body: new URLSearchParams({
    grant_type: 'authorization_code',
    code: authCode,
    redirect_uri: uri,
    client_id: id,
    code_verifier: verifier  // <-- NEW
  })
});
```

## Step 2: Remove Implicit Grant

### Identify Implicit Clients

```bash
# Find clients using response_type=token
curl https://api.ggid.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  jq '.items[] | select(.response_types | contains(["token"]))'
```

### Migrate SPA from Implicit to Code+PKCE

```javascript
// BEFORE: Implicit (token in URL fragment)
window.location = `${issuer}/authorize?response_type=token&client_id=${id}`;
// Token returned in URL: #access_token=eyJhbG...

// AFTER: Authorization Code + PKCE
window.location = `${issuer}/authorize?response_type=code&client_id=${id}
  &code_challenge=${challenge}&code_challenge_method=S256`;
// Code returned in URL: ?code=AUTH_CODE
// Exchange code for token server-side or via BFF pattern
```

## Step 3: Remove ROPC Grant

### Migrate from ROPC to Code+PKCE

```bash
# BEFORE: ROPC
curl -X POST ${issuer}/token \
  -d "grant_type=password&username=alice&password=pass&client_id=xxx"

# AFTER: GGID auth service login (not OAuth grant)
curl -X POST https://api.ggid.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"alice","password":"pass"}'
# Returns JWT directly
```

## Step 4: Enforce Exact Redirect URI Matching

### Audit Current URIs

```bash
curl https://api.ggid.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  jq '.items[] | {client_id, redirect_uris}'
```

### Fix Wildcard URIs

```yaml
# BEFORE (not allowed in OAuth 2.1)
redirect_uris:
  - "https://*.example.com/callback"  # Wildcard
  - "https://app.example.com/*"        # Trailing wildcard

# AFTER (exact match required)
redirect_uris:
  - "https://app.example.com/callback"
  - "https://staging.example.com/callback"
```

## Step 5: Enable Refresh Token Rotation

```bash
curl -X PUT https://api.ggid.example.com/api/v1/oauth/clients/$CLIENT_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "refresh_token_rotation": true,
    "refresh_token_reuse_detection": true
  }'
```

With rotation, each refresh token is single-use. If a reused token is detected, the entire token family is revoked (theft detection).

## Step 6: Verify `iss` Parameter

GGID includes `iss` in authorization responses (since commit 72edaa5). Verify your client checks it:

```javascript
// In callback handler
const params = new URLSearchParams(window.location.search);
const code = params.get('code');
const iss = params.get('iss');

if (iss !== expectedIssuer) {
  throw new Error('Issuer mismatch — possible mix-up attack');
}
```

## Step 7: Adopt DPoP (Recommended)

DPoP binds tokens to a client key pair, preventing token theft:

```javascript
// Generate DPoP key
const keyPair = await crypto.subtle.generateKey(
  { name: 'ECDSA', namedCurve: 'P-256' },
  false, ['sign', 'verify']
);

// Create DPoP proof for each request
const proof = await createDPoPProof(keyPair, 'POST', url);
fetch(url, {
  headers: {
    'Authorization': `DPoP ${accessToken}`,
    'DPoP': proof
  }
});
```

## Step 8: Adopt PAR (Recommended)

Pushed Authorization Requests move auth params from URL to server:

```bash
curl -X POST https://api.ggid.example.com/oauth/par \
  -H "Authorization: Basic $(base64 client:secret)" \
  -d "response_type=code&client_id=xxx&redirect_uri=xxx&code_challenge=xxx&state=xxx"

# Response: { "request_uri": "urn:ietf:params:oauth:request_uri:xxx" }

# Redirect with just request_uri
window.location = `${issuer}/authorize?client_id=${id}&request_uri=${requestUri}`;
```

## Migration Timeline

| Phase | Duration | Tasks |
|-------|----------|-------|
| 1. Audit | 1 week | Inventory clients, identify non-compliant |
| 2. PKCE | 2 weeks | Add PKCE to all clients |
| 3. Remove implicit | 2 weeks | Migrate SPAs to code+PKCE |
| 4. Redirect URIs | 1 week | Fix wildcards |
| 5. Refresh rotation | 1 week | Enable single-use |
| 6. Test | 1 week | Full end-to-end testing |
| 7. Deploy | Ongoing | Gradual rollout |

## Post-Migration Checklist

- [ ] All clients use PKCE (S256)
- [ ] No implicit grant clients
- [ ] No ROPC grant usage
- [ ] All redirect URIs exact match
- [ ] Refresh token rotation enabled
- [ ] `iss` parameter verified
- [ ] DPoP evaluated (optional)
- [ ] PAR evaluated (optional)
- [ ] End-to-end OAuth flow tested

## See Also

- [OAuth 2.1 Changes](../research/oauth-2.1-changes.md)
- [OAuth API Reference](../api/oauth-api.md)
- [Open Banking FAPI](../research/open-banking-fapi.md)
- [Token Binding](../research/token-binding.md)
