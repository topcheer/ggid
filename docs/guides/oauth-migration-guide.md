# OAuth 2.0 to 2.1 Migration Guide

Step-by-step migration from OAuth 2.0 to OAuth 2.1 in GGID.

> **Related**: [OAuth 2.1 Changes](../research/oauth-2.1-changes.md), [OAuth 2.1 Migration](oauth-2-1-migration.md)

## What Changes

| Feature | OAuth 2.0 | OAuth 2.1 |
|---------|-----------|-----------|
| PKCE | Optional | **Mandatory** |
| Implicit grant | Supported | **Removed** |
| Password grant (ROPC) | Supported | **Removed** |
| Redirect URI | Prefix match | **Exact match only** |
| Refresh rotation | Optional | **Required (public clients)** |
| `iss` parameter | Optional | **Mandatory** |

## Step 1: Audit Current Clients

```bash
curl https://api.ggid.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq '.items[] | {client_id, grants, response_types, redirect_uris}'
```

Flag clients using: `response_type=token` (implicit), `grant_type=password` (ROPC), wildcard redirect URIs.

## Step 2: Enable PKCE (All Clients)

```bash
curl -X PUT https://api.ggid.example.com/api/v1/oauth/clients/$CID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"require_pkce":true,"pkce_methods":["S256"]}'
```

Update app code:
```javascript
const verifier = generateRandomString(64);
const challenge = base64url(sha256(verifier));
// Add to authorize URL: code_challenge + code_challenge_method=S256
// Add to token request: code_verifier
```

## Step 3: Remove Implicit Grant

Migrate SPAs from `response_type=token` to `response_type=code` + PKCE.

## Step 4: Remove ROPC

Replace `grant_type=password` with GGID's auth login endpoint (`POST /api/v1/auth/login`).

## Step 5: Fix Redirect URIs

```yaml
# BEFORE (wildcard - not allowed)
redirect_uris: ["https://*.example.com/callback"]
# AFTER (exact)
redirect_uris: ["https://app.example.com/callback"]
```

## Step 6: Enable Refresh Rotation

```bash
curl -X PUT https://api.ggid.example.com/api/v1/oauth/clients/$CID \
  -d '{"refresh_token_rotation":true,"refresh_token_reuse_detection":true}'
```

## Step 7: Verify `iss` Parameter

GGID includes `iss` in auth responses. Client must verify:
```javascript
const iss = params.get('iss');
if (iss !== expectedIssuer) throw new Error('Mix-up attack');
```

## Rollback Strategy

1. Keep OAuth 2.0 config as fallback for 30 days
2. Monitor error rates after each step
3. If breakage > 1%: revert client config, investigate
4. Communicate timeline to integration partners

## Timeline

| Phase | Duration |
|-------|----------|
| Audit | 1 week |
| PKCE | 2 weeks |
| Remove implicit/ROPC | 2 weeks |
| Redirect URI fixes | 1 week |
| Refresh rotation | 1 week |
| Testing | 1 week |
| Deploy | Ongoing |

## See Also

- [OAuth 2.1 Changes](../research/oauth-2.1-changes.md)
- [OAuth 2.1 Migration](oauth-2-1-migration.md)
- [OAuth API](../api/oauth.md)
