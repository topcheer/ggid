# OAuth 2.1 Migration Guide

This guide covers migrating from OAuth 2.0 to OAuth 2.1 in GGID, including change analysis, compatibility matrix, phased migration plan, and testing strategy.

## OAuth 2.0 → 2.1 Changes

### Summary of Changes

| Feature | OAuth 2.0 | OAuth 2.1 | Impact |
|---|---|---|---|
| Implicit grant | Allowed | **Removed** | SPAs must use authorization code + PKCE |
| Password grant | Allowed | **Removed** | Use authorization code or device flow |
| PKCE | Optional | **Mandatory** for all clients | All clients must implement PKCE |
| Redirect URI | Wildcard/prefix match | **Exact match only** | No wildcard redirect URIs |
| Refresh token rotation | Optional | **Mandatory** (for public clients) | All public clients must rotate |
| Token introspection | RFC 7662 (separate) | **Integrated** | Built-in introspection endpoint |
| State parameter | Recommended | **Mandatory** | CSRF protection enforced |
| CORS | Not specified | **Origin-specific** | No wildcard CORS for token endpoint |

### Detailed Change Analysis

#### 1. Implicit Grant Removed

**Before**: SPAs used `response_type=token` to get access tokens directly from the authorization endpoint.

**After**: SPAs must use `response_type=code` with PKCE, exchanging the code at the token endpoint.

```javascript
// Before (OAuth 2.0 implicit)
window.location = `/authorize?response_type=token&client_id=...`

// After (OAuth 2.1 with PKCE)
const codeVerifier = generateRandomString(128);
const codeChallenge = base64url(sha256(codeVerifier));
window.location = `/authorize?response_type=code&client_id=...&code_challenge=${codeChallenge}&code_challenge_method=S256`;
// Then exchange code at token endpoint with code_verifier
```

#### 2. Password Grant Removed

**Before**: `grant_type=password` allowed clients to collect user credentials directly.

**After**: Use authorization code flow (redirect) or device authorization flow (for input-constrained devices).

#### 3. PKCE Mandatory

**Before**: PKCE (RFC 7636) was optional, recommended for public clients.

**After**: All clients, including confidential clients, must send `code_challenge` and `code_challenge_method=S256`.

#### 4. Exact Redirect URI Match

**Before**: Some implementations allowed prefix matching or wildcards.

**After**: Redirect URI must match exactly what was registered. No path parameters, no trailing slash variations.

## GGID Compatibility Matrix

| GGID Feature | OAuth 2.0 | OAuth 2.1 | Status |
|---|---|---|---|
| Authorization Code + PKCE | Supported | Required | Already supported |
| Implicit grant | Supported | Not supported | Deprecation flag available |
| Password grant | Not supported | Not supported | Never implemented |
| Client Credentials | Supported | Supported | Compatible |
| Refresh Token | Supported | Supported (rotation mandatory) | Rotation configurable |
| Device Code | Supported | Supported | Compatible |
| Token Introspection | Supported (RFC 7662) | Integrated | Compatible |
| Token Revocation | Supported (RFC 7009) | Supported | Compatible |
| DPoP | Supported | Supported | Compatible |
| PAR (RFC 9126) | Supported | Supported | Compatible |
| JAR (RFC 9101) | Supported | Supported | Compatible |

## Phased Migration Plan

### Phase 1: PKCE Enforcement (Weeks 1-2)

**Goal**: All clients must use PKCE.

```yaml
oauth:
  pkce:
    required: true
    method: "S256"  # Plain not allowed
    transition_period: 30d  # Log warnings for non-PKCE requests
```

**Actions**:
1. Enable PKCE requirement in GGID config
2. Update all SDK examples to include PKCE
3. Notify client developers of upcoming requirement
4. Monitor logs for clients not sending PKCE
5. After transition period, reject non-PKCE requests

**Verification**:
```bash
# Test: Authorization without PKCE is rejected
curl "/authorize?response_type=code&client_id=test&redirect_uri=https://app.example.com/cb"
# Expected: 400 error "code_challenge required"

# Test: Authorization with PKCE works
curl "/authorize?response_type=code&client_id=test&redirect_uri=https://app.example.com/cb&code_challenge=...&code_challenge_method=S256"
# Expected: 302 redirect with code
```

### Phase 2: Deprecate Implicit Grant (Weeks 3-4)

**Goal**: Remove implicit grant support.

```yaml
oauth:
  grants:
    implicit:
      enabled: false  # Disable
      deprecation_warning: true  # Log warning if attempted
```

**Actions**:
1. Disable implicit grant in GGID config
2. Identify all clients using `response_type=token`
3. Migrate each client to authorization code + PKCE
4. Update SDKs to remove implicit flow helpers
5. Update documentation

**Migration for SPAs**:
```javascript
// Old: implicit flow
OAuthClient.getTokenImplicit();

// New: authorization code with PKCE
OAuthClient.startAuthCodeFlow({
  pkce: true,
  redirectUri: window.location.origin + '/callback'
});
```

### Phase 3: Refresh Token Rotation (Weeks 5-6)

**Goal**: Enforce refresh token rotation for all public clients.

```yaml
oauth:
  refresh_token:
    rotation: "required"  # for public clients
    reuse_detection: true
    family_revocation_on_reuse: true
    lifetime: 7d
```

**Actions**:
1. Enable refresh token rotation (if not already)
2. Enable reuse detection and family revocation
3. Notify clients that refresh tokens are one-time use
4. Update SDKs to handle rotation automatically
5. Monitor for reuse detection events

### Phase 4: Full OAuth 2.1 Compliance (Weeks 7-8)

**Goal**: Enforce all OAuth 2.1 requirements.

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

**Actions**:
1. Enable all enforcement checks
2. Remove deprecated grant types
3. Update OpenID Connect metadata to reflect 2.1
4. Update security documentation
5. Run full compliance test suite

## Client Impact Analysis

### Web Applications (Confidential Clients)

| Change | Impact | Migration Effort |
|---|---|---|
| PKCE mandatory | Low — add code_challenge | Minimal |
| Exact redirect URI | Low — verify registered URIs | Minimal |
| Refresh rotation | Low — SDK handles automatically | Minimal |

### Single Page Applications (Public Clients)

| Change | Impact | Migration Effort |
|---|---|---|
| Implicit removed | **High** — rewrite auth flow | Moderate |
| PKCE mandatory | Medium — implement PKCE | Low |
| Refresh rotation | Medium — handle new tokens | Low |

### Mobile Apps (Public Clients)

| Change | Impact | Migration Effort |
|---|---|---|
| PKCE mandatory | Low — most already use PKCE | Minimal |
| Exact redirect URI | Low — use custom scheme | Minimal |
| Refresh rotation | Low — SDK handles | Minimal |

### Server-to-Server (Confidential Clients)

| Change | Impact | Migration Effort |
|---|---|---|
| Client credentials unchanged | None | None |
| PKCE for auth code | Low | Minimal |

## Testing Strategy

### Per-Phase Testing

Each phase has a gate test that must pass before proceeding:

```bash
# Phase 1 gate: PKCE enforcement
go test ./services/oauth/internal/service/ -run TestPKCEEnforcement

# Phase 2 gate: Implicit rejection
go test ./services/oauth/internal/service/ -run TestImplicitRejected

# Phase 3 gate: Refresh rotation
go test ./services/oauth/internal/service/ -run TestRefreshRotation

# Phase 4 gate: Full compliance
go test ./services/oauth/... -run TestOAuth21Compliance
```

### Client Compatibility Testing

For each registered client, test:

1. Authorization request succeeds with PKCE
2. Token exchange succeeds with code_verifier
3. Refresh token rotation works
4. Redirect URI exact match enforced
5. State parameter validated

### Rollback Plan

If migration causes issues:

```yaml
oauth:
  version: "2.0"  # Revert to 2.0
  enforce:
    pkce: false      # Make PKCE optional again
    no_implicit: false  # Re-enable implicit
```

**Rollback triggers**:
- >5% of client auth failures after enforcement
- Critical client unable to migrate
- Security regression detected

### Monitoring

Track these metrics during migration:

| Metric | Target | Alert |
|---|---|---|
| Auth success rate | >95% | <90% |
| PKCE adoption rate | 100% by Phase 1 end | <80% after 2 weeks |
| Implicit grant usage | 0 by Phase 2 end | >0 after Phase 2 |
| Refresh rotation compliance | 100% | <95% |
| Reuse detection events | Baseline | Spike (>10/day) |

## Best Practices

1. **Communicate early** — notify all client developers 30 days before each phase
2. **Monitor between phases** — track adoption metrics and failure rates
3. **Provide SDK updates** — update Go/Node/Java SDKs before enforcement
4. **Document breaking changes** — clear migration guide for each client type
5. **Test with real clients** — don't just test with synthetic data
6. **Have rollback ready** — be prepared to revert if critical clients break
7. **Log deprecation warnings** — help clients identify what needs updating
8. **Phase gradually** — 2-week phases give clients time to adapt