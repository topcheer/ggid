# OIDC Discovery Metadata Security for IAM Systems

> **Research Document** — GGID IAM Suite  
> Topic: Security analysis of OpenID Connect Discovery metadata, JWKS handling, and RP-side validation  
> Audience: Security engineers, platform developers, and relying party integrators

---

## Table of Contents

1. [Discovery Metadata Structure](#1-discovery-metadata-structure)
2. [Metadata Validation for RPs](#2-metadata-validation-for-rps)
3. [JWKS URI Security](#3-jwks-uri-security)
4. [Discovery Cache Poisoning](#4-discovery-cache-poisoning)
5. [Metadata Tampering Detection](#5-metadata-tampering-detection)
6. [Signing Key Validation](#6-signing-key-validation)
7. [Dynamic Configuration Updates](#7-dynamic-configuration-updates)
8. [Multi-Tenant Discovery](#8-multi-tenant-discovery)
9. [GGID Discovery Audit](#9-ggid-discovery-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Discovery Metadata Structure

### Purpose

The OIDC Discovery endpoint at `/.well-known/openid-configuration` (RFC 8414 / OpenID Connect
Discovery 1.0) allows Relying Parties (RPs) to dynamically learn the Provider's (OP) endpoints,
supported algorithms, and cryptographic parameters. This eliminates manual configuration but
introduces a trust boundary: if discovery metadata is compromised, the RP can be directed to
attacker-controlled endpoints.

### Standard Fields

| Field | Required | Description |
|-------|----------|-------------|
| `issuer` | Yes | OP Issuer URL — must exactly match the URL the document was fetched from |
| `authorization_endpoint` | Yes | OAuth 2.0 authorization endpoint URL |
| `token_endpoint` | Yes | OAuth 2.0 token endpoint URL |
| `userinfo_endpoint` | No | UserInfo endpoint URL |
| `jwks_uri` | Yes | URL of the OP's JSON Web Key Set |
| `scopes_supported` | No | List of supported scopes |
| `response_types_supported` | Yes | List of supported response_type values |
| `grant_types_supported` | No | List of supported grant_type values |
| `subject_types_supported` | Yes | `public` or `pairwise` |
| `id_token_signing_alg_values_supported` | Yes | JWS signing algorithms supported for ID Tokens |
| `claims_supported` | No | List of claim names that may appear in ID Tokens |
| `token_endpoint_auth_methods_supported` | No | Supported client authentication methods |
| `code_challenge_methods_supported` | No | PKCE code challenge methods |
| `revocation_endpoint` | No | Token revocation endpoint |
| `introspection_endpoint` | No | Token introspection endpoint |
| `end_session_endpoint` | No | RP-initiated logout endpoint |

### Go: Discovery Metadata Struct

```go
package discovery

// DiscoveryMetadata represents the OIDC .well-known/openid-configuration document.
type DiscoveryMetadata struct {
    Issuer                 string   `json:"issuer"`
    AuthorizationEndpoint  string   `json:"authorization_endpoint"`
    TokenEndpoint          string   `json:"token_endpoint"`
    UserInfoEndpoint       string   `json:"userinfo_endpoint,omitempty"`
    JWKSURI                string   `json:"jwks_uri"`
    RevocationEndpoint     string   `json:"revocation_endpoint,omitempty"`
    IntrospectionEndpoint  string   `json:"introspection_endpoint,omitempty"`
    EndSessionEndpoint     string   `json:"end_session_endpoint,omitempty"`

    ResponseTypesSupported            []string `json:"response_types_supported"`
    GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
    SubjectTypesSupported             []string `json:"subject_types_supported"`
    IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
    ScopesSupported                   []string `json:"scopes_supported,omitempty"`
    ClaimsSupported                   []string `json:"claims_supported,omitempty"`
    TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
    CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
    CheckSessionIFrame                string   `json:"check_session_iframe,omitempty"`
    BackchannelLogoutSupported        bool     `json:"backchannel_logout_supported,omitempty"`
    RequestParameterSupported         bool     `json:"request_parameter_supported,omitempty"`
    RequestURIParameterSupported      bool     `json:"request_uri_parameter_supported,omitempty"`
    RequireRequestURIRegistration     bool     `json:"require_request_uri_registration,omitempty"`
    OPPolicyURI                       string   `json:"op_policy_uri,omitempty"`
    OPTermsURI                        string   `json:"op_tos_uri,omitempty"`
}
```

### Go: Discovery Handler

```go
package discovery

import (
    "encoding/json"
    "net/http"
)

func DiscoveryHandler(cfg *DiscoveryMetadata) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Only respond to exact path — no query params accepted
        if r.URL.RawQuery != "" {
            http.Error(w, "query parameters not allowed", http.StatusBadRequest)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        // Allow CDN caching but require revalidation to prevent stale metadata
        w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
        w.Header().Set("Access-Control-Allow-Origin", "*") // Discovery is public info

        json.NewEncoder(w).Encode(cfg)
    }
}
```

**Key design decisions:**
- Query parameters are rejected to prevent cache-key injection (see Section 4).
- `Cache-Control: must-revalidate` prevents serving stale metadata from intermediary caches.
- CORS is open because discovery metadata is intentionally public.

---

## 2. Metadata Validation for RPs

### Why Validation Matters

An RP that blindly trusts discovery metadata is vulnerable to:
- **Issuer impersonation** — metadata claims a different issuer than the URL it was fetched from.
- **Algorithm downgrade attacks** — metadata advertises `none` or `HS256` in place of secure RSA/ECDSA algorithms.
- **Endpoint redirection** — metadata points authorization/token endpoints to attacker-controlled URLs.
- **Transport downgrade** — metadata fetched over HTTP can be modified in transit.

### Validation Rules

1. **Issuer URL match**: The `issuer` field must exactly match the URL the document was fetched from (after removing the `/.well-known/openid-configuration` path suffix). RFC 8414 Section 3: "The Issuer value must exactly match the value of the `iss` (issuer) claim in ID Tokens."

2. **HTTPS-only**: All URLs in the metadata must use `https://`. HTTP must be rejected unconditionally. This includes `jwks_uri`, `authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, etc.

3. **Required fields**: `issuer`, `authorization_endpoint`, `token_endpoint`, `jwks_uri`, `response_types_supported`, `subject_types_supported`, `id_token_signing_alg_values_supported` must be present and non-empty.

4. **Safe algorithms**: `id_token_signing_alg_values_supported` must not contain:
   - `none` — unsigned tokens are never acceptable.
   - `HS256`, `HS384`, `HS512` — HMAC with shared secrets is vulnerable to key confusion attacks when the RP also has the JWKS public key.
   - `RS1`, `PS1`, `ES1` — SHA-1 algorithms are cryptographically weak.

5. **Same-origin endpoints**: All endpoint URLs should share the same scheme and host as the issuer.

### Dangerous Algorithms

| Algorithm | Risk |
|-----------|------|
| `none` | No signature — complete bypass of token integrity |
| `HS256/384/512` | Key confusion: RP may verify with public key as HMAC key |
| `RS1/PS1` | SHA-1 collision attacks weaken signature |
| `ES256K` (if unsupported) | Non-standard curve, implementation-dependent |
| `EdDSA` (if unsupported) | Library must support Ed25519 properly |

### Go: RP-Side Metadata Validation

```go
package rp

import (
    "fmt"
    "net/url"
    "strings"
)

// UnsafeAlgorithms are never acceptable for ID Token signing.
var unsafeAlgorithms = map[string]bool{
    "none": true,
    "HS256": true, "HS384": true, "HS512": true, // HMAC
    "RS1": true, "PS1": true,                    // SHA-1 based
}

// ValidateDiscoveryMetadata validates OIDC discovery metadata fetched from the given issuer URL.
func ValidateDiscoveryMetadata(issuerURL string, meta *DiscoveryMetadata) error {
    // 1. Validate issuer URL itself is HTTPS
    issuer, err := url.Parse(issuerURL)
    if err != nil {
        return fmt.Errorf("invalid issuer URL: %w", err)
    }
    if issuer.Scheme != "https" {
        return fmt.Errorf("issuer must use HTTPS, got %s", issuer.Scheme)
    }
    if issuer.Path != "" && issuer.Path != "/" {
        return fmt.Errorf("issuer URL must have no path component, got %s", issuer.Path)
    }

    // 2. Issuer field must match the fetch URL exactly
    if meta.Issuer != strings.TrimSuffix(issuerURL, "/") {
        return fmt.Errorf("issuer mismatch: expected %q, got %q", issuerURL, meta.Issuer)
    }

    // 3. All endpoint URLs must be HTTPS and on the same host
    endpoints := []struct{ name, val string }{
        {"authorization_endpoint", meta.AuthorizationEndpoint},
        {"token_endpoint", meta.TokenEndpoint},
        {"userinfo_endpoint", meta.UserInfoEndpoint},
        {"jwks_uri", meta.JWKSURI},
        {"revocation_endpoint", meta.RevocationEndpoint},
        {"introspection_endpoint", meta.IntrospectionEndpoint},
    }
    for _, ep := range endpoints {
        if ep.val == "" {
            continue // optional field
        }
        u, err := url.Parse(ep.val)
        if err != nil {
            return fmt.Errorf("invalid %s: %w", ep.name, err)
        }
        if u.Scheme != "https" {
            return fmt.Errorf("%s must use HTTPS, got %s", ep.name, u.Scheme)
        }
        if u.Host != issuer.Host {
            return fmt.Errorf("%s host mismatch: expected %s, got %s",
                ep.name, issuer.Host, u.Host)
        }
    }

    // 4. Required fields present
    if meta.JWKSURI == "" {
        return fmt.Errorf("jwks_uri is required")
    }
    if len(meta.ResponseTypesSupported) == 0 {
        return fmt.Errorf("response_types_supported is required")
    }
    if len(meta.SubjectTypesSupported) == 0 {
        return fmt.Errorf("subject_types_supported is required")
    }
    if len(meta.IDTokenSigningAlgValuesSupported) == 0 {
        return fmt.Errorf("id_token_signing_alg_values_supported is required")
    }

    // 5. Algorithm safety check
    for _, alg := range meta.IDTokenSigningAlgValuesSupported {
        if unsafeAlgorithms[alg] {
            return fmt.Errorf("unsafe signing algorithm advertised: %s", alg)
        }
    }

    return nil
}
```

---

## 3. JWKS URI Security

### Threat Model

The JWKS endpoint exposes public signing keys that RPs use to verify tokens. Attack vectors:

1. **MITM on JWKS fetch** — If JWKS is fetched over HTTP, an attacker can substitute their own keys.
2. **Unknown key ID (kid) injection** — An attacker who controls a subset of the JWKS can inject a key with a `kid` that the RP has never seen.
3. **DoS via rapid refresh** — An attacker can trigger excessive JWKS refreshes by sending tokens with rotating unknown `kid` values.
4. **Key confusion** — Mixing RSA and HMAC keys in the same JWKS can lead to algorithm confusion if the RP doesn't validate `kty`.

### Security Requirements

| Requirement | Implementation |
|-------------|----------------|
| HTTPS-only JWKS URI | Reject `http://` in `jwks_uri` |
| Cache with Cache-Control | Respect `max-age` from response headers |
| Refresh on unknown kid | Only after cache expiry; rate-limited |
| Rate-limit refreshes | Maximum 1 refresh per 5 minutes per issuer |
| Validate key format | Check `kty`, `use`, `alg`, `n`/`e` format |
| Reject symmetric keys | No `oct` key type in public JWKS |

### Go: Secure JWKS Fetcher with Caching

```go
package jwks

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    "net/http"
    "net/url"
    "sync"
    "crypto/rsa"
    "time"
)

// JWK represents a single JSON Web Key.
type JWK struct {
    KTY string `json:"kty"`
    Use string `json:"use"`
    Alg string `json:"alg"`
    KID string `json:"kid"`
    N   string `json:"n,omitempty"`
    E   string `json:"e,omitempty"`
    X   string `json:"x,omitempty"`
    Y   string `json:"y,omitempty"`
    Crv string `json:"crv,omitempty"`
    K   string `json:"k,omitempty"` // symmetric — must reject
}

type JWKSResponse struct {
    Keys []JWK `json:"keys"`
}

// SecureJWKSClient fetches and caches JWKS with security controls.
type SecureJWKSClient struct {
    jwksURI    string
    httpClient *http.Client

    mu          sync.RWMutex
    cachedKeys  map[string]*rsa.PublicKey
    cachedAt    time.Time
    cacheExpiry time.Duration

    lastFetch     time.Time
    minRefreshGap time.Duration // rate limit
}

func NewSecureJWKSClient(jwksURI string) (*SecureJWKSClient, error) {
    u, err := url.Parse(jwksURI)
    if err != nil {
        return nil, fmt.Errorf("invalid jwks_uri: %w", err)
    }
    if u.Scheme != "https" {
        return nil, fmt.Errorf("jwks_uri must use HTTPS, got %s", u.Scheme)
    }

    return &SecureJWKSClient{
        jwksURI:       jwksURI,
        httpClient:    &http.Client{Timeout: 10 * time.Second},
        cachedKeys:    make(map[string]*rsa.PublicKey),
        cacheExpiry:   15 * time.Minute,
        minRefreshGap: 5 * time.Minute,
    }, nil
}

// GetKey returns the RSA public key for the given kid, refreshing if necessary.
func (c *SecureJWKSClient) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
    // Try cache first
    c.mu.RLock()
    if key, ok := c.cachedKeys[kid]; ok {
        c.mu.RUnlock()
        return key, nil
    }
    if time.Since(c.cachedAt) < c.cacheExpiry {
        c.mu.RUnlock()
        return nil, fmt.Errorf("key %q not found in valid cache", kid)
    }
    c.mu.RUnlock()

    // Rate-limit refresh attempts
    c.mu.Lock()
    if time.Since(c.lastFetch) < c.minRefreshGap {
        c.mu.Unlock()
        return nil, fmt.Errorf("rate limited: JWKS refresh attempted too recently")
    }
    c.lastFetch = time.Now()
    c.mu.Unlock()

    if err := c.refresh(ctx); err != nil {
        return nil, fmt.Errorf("jwks refresh: %w", err)
    }

    // Re-check cache after refresh
    c.mu.RLock()
    defer c.mu.RUnlock()
    if key, ok := c.cachedKeys[kid]; ok {
        return key, nil
    }
    return nil, fmt.Errorf("key %q not found after refresh", kid)
}

func (c *SecureJWKSClient) refresh(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.jwksURI, nil)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
    }

    body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
    if err != nil {
        return err
    }

    var jwks JWKSResponse
    if err := json.Unmarshal(body, &jwks); err != nil {
        return fmt.Errorf("invalid JWKS JSON: %w", err)
    }

    // Validate each key and build new cache
    newKeys := make(map[string]*rsa.PublicKey)
    for _, key := range jwks.Keys {
        pub, err := validateAndConvertRSAKey(key)
        if err != nil {
            continue // skip invalid keys silently
        }
        newKeys[key.KID] = pub
    }

    if len(newKeys) == 0 {
        return fmt.Errorf("no valid RSA signing keys in JWKS")
    }

    // Parse Cache-Control for max-age
    maxAge := parseMaxAge(resp.Header.Get("Cache-Control"))
    if maxAge > 0 {
        c.cacheExpiry = maxAge
    }

    c.mu.Lock()
    c.cachedKeys = newKeys
    c.cachedAt = time.Now()
    c.mu.Unlock()

    return nil
}

func validateAndConvertRSAKey(key JWK) (*rsa.PublicKey, error) {
    // Reject symmetric keys
    if key.KTY == "oct" {
        return nil, fmt.Errorf("symmetric keys are not allowed in public JWKS")
    }
    // Only RSA keys supported
    if key.KTY != "RSA" {
        return nil, fmt.Errorf("unsupported key type: %s", key.KTY)
    }
    // Must be a signing key
    if key.Use != "" && key.Use != "sig" {
        return nil, fmt.Errorf("key use must be 'sig', got %s", key.Use)
    }
    // Algorithm must be RSA-based
    if key.Alg != "" && !strings.HasPrefix(key.Alg, "RS") && !strings.HasPrefix(key.Alg, "PS") {
        return nil, fmt.Errorf("key algorithm must be RSA-based, got %s", key.Alg)
    }
    // N and E must be present
    if key.N == "" || key.E == "" {
        return nil, fmt.Errorf("missing RSA key parameters n or e")
    }

    nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
    if err != nil {
        return nil, fmt.Errorf("invalid n encoding: %w", err)
    }
    eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
    if err != nil {
        return nil, fmt.Errorf("invalid e encoding: %w", err)
    }

    n := new(big.Int).SetBytes(nBytes)
    e := new(big.Int).SetBytes(eBytes)

    // Minimum key size: RSA 2048 bits
    if n.BitLen() < 2048 {
        return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048)", n.BitLen())
    }

    return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func parseMaxAge(cacheControl string) time.Duration {
    // Parse "max-age=3600" from Cache-Control header
    parts := strings.Split(cacheControl, ",")
    for _, part := range parts {
        part = strings.TrimSpace(part)
        if strings.HasPrefix(part, "max-age=") {
            seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
            if err == nil && seconds > 0 {
                return time.Duration(seconds) * time.Second
            }
        }
    }
    return 0
}
```

---

## 4. Discovery Cache Poisoning

### Attack Vectors

#### CDN Cache Poisoning via Query Parameters

If the CDN includes query parameters in the cache key, an attacker can inject a poisoned
discovery response:

```
GET /.well-known/openid-configuration?callback=evil.com
```

If the server reflects the callback parameter in the response body, and the CDN caches the
response keyed by the full URL, subsequent requests with the same query string receive the
poisoned version.

**Defense**: Discovery handlers must reject any request with query parameters (as shown in
Section 1's handler). Additionally, set `Vary: Accept` to prevent content-type confusion.

#### MITM on HTTP Discovery

If an RP initially fetches discovery over HTTP (e.g., as a redirect from `http://` to
`https://`), a MITM can intercept and modify the metadata before the upgrade. The modified
metadata can point all endpoints to attacker-controlled HTTPS URLs.

**Defense**: Never follow HTTP discovery. The RP must construct the HTTPS discovery URL
directly: `https://{issuer}/.well-known/openid-configuration`.

#### DNS Rebinding

After the RP resolves the issuer's IP address, the attacker changes the DNS record to point
to an attacker-controlled server. The RP then connects to the attacker's server, which serves
malicious metadata.

**Defense**:
- Pin the IP address from the first DNS resolution for subsequent connections.
- Validate the TLS certificate against the original hostname (standard TLS does this, but
  only if the RP doesn't disable certificate validation).
- Consider HTTP Origin/Host header validation on the server side.

### Go: Discovery Cache with Validation

```go
package discovery

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

// CachedDiscovery serves discovery metadata with integrity tracking.
type CachedDiscovery struct {
    metadata    *DiscoveryMetadata
    metadataHash string // SHA-256 of serialized metadata
    fetchedAt   time.Time
    mu          sync.RWMutex
    maxAge      time.Duration

    // Integrity monitoring
    previousHash string
    onChange     func(oldHash, newHash string, oldMeta, newMeta *DiscoveryMetadata)
}

func NewCachedDiscovery(maxAge time.Duration) *CachedDiscovery {
    return &CachedDiscovery{
        maxAge: maxAge,
    }
}

// FetchAndValidate fetches discovery metadata and validates it against the expected issuer.
func (c *CachedDiscovery) FetchAndValidate(issuerURL string) error {
    // Ensure HTTPS
    if !strings.HasPrefix(issuerURL, "https://") {
        return fmt.Errorf("discovery URL must use HTTPS")
    }

    resp, err := http.Get(issuerURL)
    if err != nil {
        return fmt.Errorf("fetch discovery: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("discovery returned %d", resp.StatusCode)
    }

    var meta DiscoveryMetadata
    if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
        return fmt.Errorf("decode discovery: %w", err)
    }

    // Validate metadata
    if err := ValidateDiscoveryMetadata(issuerURL, &meta); err != nil {
        return fmt.Errorf("metadata validation failed: %w", err)
    }

    // Compute hash for integrity monitoring
    raw, _ := json.Marshal(&meta)
    hash := sha256.Sum256(raw)
    hashHex := hex.EncodeToString(hash[:])

    c.mu.Lock()
    c.previousHash = c.metadataHash
    c.metadata = &meta
    c.metadataHash = hashHex
    c.fetchedAt = time.Now()

    // Detect changes
    if c.previousHash != "" && c.previousHash != hashHex && c.onChange != nil {
        c.onChange(c.previousHash, hashHex, c.metadata, &meta)
    }
    c.mu.Unlock()

    return nil
}

// Metadata returns cached metadata if fresh, or triggers a re-fetch.
func (c *CachedDiscovery) Metadata() (*DiscoveryMetadata, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    if c.metadata == nil {
        return nil, fmt.Errorf("no cached metadata — fetch first")
    }
    if time.Since(c.fetchedAt) > c.maxAge {
        return c.metadata, fmt.Errorf("metadata expired — re-fetch required")
    }
    return c.metadata, nil
}
```

---

## 5. Metadata Tampering Detection

### Detection Strategies

1. **Hash comparison across fetches**: Store a SHA-256 hash of the discovery metadata. If
   the hash changes between fetches, alert and log the diff. Non-trivial changes to endpoint
   URLs or algorithms warrant investigation.

2. **RFC 8414 signed metadata**: OAuth 2.0 Authorization Server Metadata (RFC 8414) allows
   signed metadata via `signed_metadata` field containing a JWS. The RP verifies the JWS
   using a pre-configured trust anchor (e.g., from OIDC Federation).

3. **Out-of-band verification**: Critical metadata fields (especially `jwks_uri` and
   `id_token_signing_alg_values_supported`) can be pinned at configuration time. Discovery
   metadata is fetched dynamically but compared against pinned values.

### Go: Metadata Integrity Monitor

```go
package monitoring

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
)

// MetadataChangeAlert describes a suspicious change in discovery metadata.
type MetadataChangeAlert struct {
    Timestamp   time.Time
    Field       string
    OldValue    any
    NewValue    any
    Severity    string // "info", "warning", "critical"
    Description string
}

// MonitorMetadataChanges compares old and new discovery metadata and generates alerts.
func MonitorMetadataChanges(old, new *DiscoveryMetadata) []MetadataChangeAlert {
    var alerts []MetadataChangeAlert

    // Critical: issuer change
    if old.Issuer != new.Issuer {
        alerts = append(alerts, MetadataChangeAlert{
            Timestamp: time.Now(),
            Field:     "issuer",
            OldValue:  old.Issuer,
            NewValue:  new.Issuer,
            Severity:  "critical",
            Description: "Issuer URL changed — possible metadata tampering",
        })
    }

    // Critical: algorithm downgrade
    oldAlgs := toSet(old.IDTokenSigningAlgValuesSupported)
    newAlgs := toSet(new.IDTokenSigningAlgValuesSupported)
    for alg := range newAlgs {
        if !oldAlgs[alg] {
            severity := "warning"
            if alg == "none" || alg == "HS256" {
                severity = "critical"
            }
            alerts = append(alerts, MetadataChangeAlert{
                Timestamp: time.Now(),
                Field:     "id_token_signing_alg_values_supported",
                OldValue:  old.IDTokenSigningAlgValuesSupported,
                NewValue:  new.IDTokenSigningAlgValuesSupported,
                Severity:  severity,
                Description: fmt.Sprintf("New signing algorithm advertised: %s", alg),
            })
        }
    }

    // Warning: endpoint changes
    endpoints := []struct{ name string; old, new string }{
        {"authorization_endpoint", old.AuthorizationEndpoint, new.AuthorizationEndpoint},
        {"token_endpoint", old.TokenEndpoint, new.TokenEndpoint},
        {"jwks_uri", old.JWKSURI, new.JWKSURI},
    }
    for _, ep := range endpoints {
        if ep.old != ep.new {
            alerts = append(alerts, MetadataChangeAlert{
                Timestamp: time.Now(),
                Field:     ep.name,
                OldValue:  ep.old,
                NewValue:  ep.new,
                Severity:  "warning",
                Description: fmt.Sprintf("Endpoint URL changed: %s -> %s", ep.old, ep.new),
            })
        }
    }

    return alerts
}

// LogAlerts writes metadata change alerts to the log.
func LogAlerts(alerts []MetadataChangeAlert) {
    for _, a := range alerts {
        switch a.Severity {
        case "critical":
            log.Printf("[CRITICAL] Metadata tampering detected: %s (%s: %v -> %v)",
                a.Description, a.Field, a.OldValue, a.NewValue)
        case "warning":
            log.Printf("[WARNING] Metadata change: %s (%s: %v -> %v)",
                a.Description, a.Field, a.OldValue, a.NewValue)
        default:
            log.Printf("[INFO] Metadata change: %s", a.Description)
        }
    }
}

func toSet(items []string) map[string]bool {
    set := make(map[string]bool, len(items))
    for _, item := range items {
        set[item] = true
    }
    return set
}
```

---

## 6. Signing Key Validation

### Key Validation Rules

| Rule | Rationale |
|------|-----------|
| `kty` must match algorithm | RS256 requires RSA key, ES256 requires EC key |
| RSA minimum 2048 bits | NIST SP 800-57 recommends 2048-bit minimum |
| ECDSA minimum P-256 | P-224 is deprecated for digital signatures |
| No `oct` keys in public JWKS | Symmetric keys should never appear in public JWKS |
| `use` must be `sig` | Encryption keys must not be used for signing |
| `kid` must be URL-safe | Prevents injection in header parsing |

### Algorithm-to-Key-Type Mapping

| Algorithm | Key Type | Minimum Size |
|-----------|----------|-------------|
| RS256/384/512 | RSA | 2048 bits |
| PS256/384/512 | RSA | 2048 bits |
| ES256 | ECDSA (P-256) | 256 bits |
| ES384 | ECDSA (P-384) | 384 bits |
| ES512 | ECDSA (P-521) | 521 bits |
| EdDSA | Ed25519 | 256 bits |

### Go: Key Validation

```go
package keys

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rsa"
    "fmt"
    "math/big"
)

// ValidateRSAKey validates an RSA public key against minimum security requirements.
func ValidateRSAKey(key *rsa.PublicKey, minBits int) error {
    if minBits == 0 {
        minBits = 2048
    }
    if key.N.BitLen() < minBits {
        return fmt.Errorf("RSA key size %d bits is below minimum %d bits",
            key.N.BitLen(), minBits)
    }
    // Check public exponent is odd and > 1
    if key.E < 3 || key.E%2 == 0 {
        return fmt.Errorf("invalid RSA public exponent: %d", key.E)
    }
    return nil
}

// ValidateECDSAKey validates an ECDSA public key against minimum curve requirements.
func ValidateECDSAKey(key *ecdsa.PublicKey) error {
    switch key.Curve {
    case elliptic.P256():
        return nil // ES256 compatible
    case elliptic.P384():
        return nil // ES384 compatible
    case elliptic.P521():
        return nil // ES512 compatible
    default:
        return fmt.Errorf("unsupported ECDSA curve: %s", key.Curve.Params().Name)
    }
}

// AlgorithmMatchesKeyType checks that the JWS algorithm is compatible with the key type.
func AlgorithmMatchesKeyType(alg, kty string) error {
    switch alg {
    case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
        if kty != "RSA" {
            return fmt.Errorf("algorithm %s requires RSA key, got %s", alg, kty)
        }
    case "ES256", "ES256K", "ES384", "ES512":
        if kty != "EC" {
            return fmt.Errorf("algorithm %s requires EC key, got %s", alg, kty)
        }
    case "EdDSA":
        if kty != "OKP" {
            return fmt.Errorf("algorithm EdDSA requires OKP key, got %s", alg, kty)
        }
    case "none":
        return fmt.Errorf("algorithm 'none' is never acceptable")
    default:
        return fmt.Errorf("unsupported algorithm: %s", alg)
    }
    return nil
}

// ValidateJWKSet validates all keys in a JWKS for safety.
func ValidateJWKSet(keys []JWK) error {
    for _, key := range keys {
        if key.KTY == "oct" {
            return fmt.Errorf("symmetric key (oct) found in public JWKS — reject immediately")
        }
        if key.Use != "" && key.Use != "sig" {
            return fmt.Errorf("key with kid %s has use=%s, expected 'sig'", key.KID, key.Use)
        }
        if key.KID == "" {
            return fmt.Errorf("key missing kid — cannot match to tokens")
        }
        if key.Alg == "none" {
            return fmt.Errorf("key with kid %s has alg=none", key.KID)
        }
    }
    return nil
}
```

---

## 7. Dynamic Configuration Updates

### Safe Update Strategy

RPs must handle metadata changes gracefully:

1. **Scheduled re-fetch**: Re-fetch discovery metadata every 24 hours (or based on
   `Cache-Control: max-age`). This picks up key rotations and endpoint changes.

2. **Algorithm downgrade detection**: If the new metadata removes RS256 and adds `none` or
   HS256, suspend token validation and alert. This is a strong signal of compromise.

3. **Endpoint URL change**: If endpoint URLs change to a different domain, require manual
   confirmation before accepting the new metadata.

4. **Key rotation**: New `kid` values in JWKS are expected during rotation. Old keys should
   be kept in cache for a grace period (e.g., 48 hours) to validate tokens issued before
   rotation.

### Go: Safe Metadata Refresh

```go
package rp

import (
    "context"
    "fmt"
    "log"
    "time"
)

// MetadataRefresher periodically refreshes discovery metadata with safety checks.
type MetadataRefresher struct {
    cache       *CachedDiscovery
    issuerURL   string
    refreshGap  time.Duration
    pinnedAlgs  []string // algorithms that must remain available
    onAlert     func(alert MetadataChangeAlert)
}

func NewMetadataRefresher(issuerURL string, cache *CachedDiscovery) *MetadataRefresher {
    return &MetadataRefresher{
        cache:      cache,
        issuerURL:  issuerURL,
        refreshGap: 24 * time.Hour,
        pinnedAlgs: []string{"RS256"}, // must always be available
        onAlert: func(a MetadataChangeAlert) {
            log.Printf("[METADATA ALERT] %s: %s", a.Severity, a.Description)
        },
    }
}

// Start begins periodic metadata refresh.
func (r *MetadataRefresher) Start(ctx context.Context) {
    ticker := time.NewTicker(r.refreshGap)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := r.safeRefresh(); err != nil {
                log.Printf("metadata refresh failed: %v — continuing with cached data", err)
            }
        }
    }
}

func (r *MetadataRefresher) safeRefresh() error {
    oldMeta, _ := r.cache.Metadata()

    if err := r.cache.FetchAndValidate(r.issuerURL); err != nil {
        return err
    }

    newMeta, err := r.cache.Metadata()
    if err != nil {
        return err
    }

    // Detect changes
    if oldMeta != nil {
        alerts := MonitorMetadataChanges(oldMeta, newMeta)
        for _, alert := range alerts {
            r.onAlert(alert)
        }
    }

    // Verify pinned algorithms are still available
    newAlgs := toSet(newMeta.IDTokenSigningAlgValuesSupported)
    for _, pinned := range r.pinnedAlgs {
        if !newAlgs[pinned] {
            return fmt.Errorf("pinned algorithm %s is no longer available — refusing update", pinned)
        }
    }

    // Check for algorithm downgrade (removal of all asymmetric algs)
    hasAsymmetric := false
    for _, alg := range newMeta.IDTokenSigningAlgValuesSupported {
        if alg == "RS256" || alg == "ES256" || alg == "PS256" {
            hasAsymmetric = true
            break
        }
    }
    if !hasAsymmetric {
        return fmt.Errorf("no asymmetric algorithms in new metadata — possible downgrade attack")
    }

    return nil
}
```

---

## 8. Multi-Tenant Discovery

### Per-Tenant Discovery Pattern

In multi-tenant IAM systems like GGID, each tenant may have:
- Different signing keys (per-tenant JWKS)
- Different OAuth clients and redirect URIs
- Different supported scopes and claims
- Different token lifetimes and policies

Two patterns for multi-tenant discovery:

#### URL-Based Tenant Resolution

```
GET /.well-known/openid-configuration/{tenant_id}
```

The tenant ID is part of the discovery URL. The issuer in the metadata includes the tenant:
```
issuer: https://auth.example.com/{tenant_id}
```

**Pros**: Clean separation, distinct issuers per tenant.  
**Cons**: Requires tenant to be in the URL path.

#### Metadata-Based Tenant Resolution

```
GET /.well-known/openid-configuration
```

A single discovery document with tenant-aware endpoints:
```
issuer: https://auth.example.com
token_endpoint: https://auth.example.com/oauth/token?tenant={tenant_id}
```

**Pros**: Single discovery document.  
**Cons**: Query-param-based tenant resolution is fragile and cache-unfriendly.

### Recommendation

URL-based tenant resolution is strongly preferred. It aligns with RFC 8414's requirement
that the issuer URL must match the document URL, and it allows clean per-tenant key isolation.

### Go: Tenant-Aware Discovery

```go
package discovery

import (
    "encoding/json"
    "net/http"
    "strings"
)

// TenantDiscoveryProvider generates per-tenant discovery metadata.
type TenantDiscoveryProvider struct {
    BaseIssuer string // e.g., "https://auth.example.com"
    KeyStore   TenantKeyStore
}

// TenantKeyStore provides per-tenant JWKS URIs and signing algorithms.
type TenantKeyStore interface {
    GetTenantConfig(tenantID string) (*TenantConfig, error)
}

type TenantConfig struct {
    TenantID        string
    JWKSURI         string
    Scopes          []string
    Claims          []string
    SigningAlgs     []string
    EndSessionURL   string
}

// TenantDiscoveryHandler serves per-tenant discovery at /.well-known/openid-configuration/{tenant}
func (p *TenantDiscoveryProvider) TenantDiscoveryHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant ID from path
        tenantID := strings.TrimPrefix(r.URL.Path, "/.well-known/openid-configuration/")
        if tenantID == "" || tenantID == r.URL.Path {
            http.Error(w, "tenant ID required", http.StatusBadRequest)
            return
        }

        // Validate tenant ID format (UUID or slug)
        if !isValidTenantID(tenantID) {
            http.Error(w, "invalid tenant ID", http.StatusBadRequest)
            return
        }

        tenantCfg, err := p.KeyStore.GetTenantConfig(tenantID)
        if err != nil {
            http.Error(w, "tenant not found", http.StatusNotFound)
            return
        }

        // Build per-tenant metadata
        issuer := p.BaseIssuer + "/" + tenantID
        meta := &DiscoveryMetadata{
            Issuer:                issuer,
            AuthorizationEndpoint: p.BaseIssuer + "/" + tenantID + "/oauth/authorize",
            TokenEndpoint:         p.BaseIssuer + "/" + tenantID + "/oauth/token",
            UserInfoEndpoint:      p.BaseIssuer + "/" + tenantID + "/oauth/userinfo",
            JWKSURI:               tenantCfg.JWKSURI,
            ResponseTypesSupported:            []string{"code"},
            GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
            SubjectTypesSupported:             []string{"public"},
            IDTokenSigningAlgValuesSupported:  tenantCfg.SigningAlgs,
            ScopesSupported:                   tenantCfg.Scopes,
            ClaimsSupported:                   tenantCfg.Claims,
            CodeChallengeMethodsSupported:     []string{"S256"},
            EndSessionEndpoint:                tenantCfg.EndSessionURL,
        }

        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Cache-Control", "private, max-age=300") // per-tenant, shorter cache
        json.NewEncoder(w).Encode(meta)
    }
}

func isValidTenantID(id string) bool {
    // Accept UUIDs or lowercase alphanumeric slugs
    if len(id) < 3 || len(id) > 64 {
        return false
    }
    for _, c := range id {
        if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '-' {
            return false
        }
    }
    return true
}
```

---

## 9. GGID Discovery Audit

### Current Implementation Review

The GGID OAuth service (`services/oauth/`) implements a basic but functional OIDC discovery
endpoint. Here is what exists and what is missing.

#### What Exists

**Discovery endpoint** (`server.go:145`):
- Served at `/.well-known/openid-configuration`
- Returns `OIDCDiscoveryConfig` struct with comprehensive fields

**Metadata fields returned** (`oauth_service.go:364-386`):

| Field | Value | Status |
|-------|-------|--------|
| `issuer` | Configurable via `s.issuer` | Present |
| `authorization_endpoint` | `{issuer}/oauth/authorize` | Present |
| `token_endpoint` | `{issuer}/oauth/token` | Present |
| `userinfo_endpoint` | `{issuer}/oauth/userinfo` | Present |
| `jwks_uri` | `{issuer}/oauth/jwks` | Present |
| `revocation_endpoint` | `{issuer}/oauth/revoke` | Present |
| `introspection_endpoint` | `{issuer}/oauth/introspect` | Present |
| `response_types_supported` | `["code", "token", "id_token"]` | Present |
| `grant_types_supported` | `["authorization_code", "refresh_token", "client_credentials"]` | Present |
| `subject_types_supported` | `["public"]` | Present |
| `id_token_signing_alg_values_supported` | `["RS256"]` | Present (safe algorithm) |
| `scopes_supported` | `["openid", "profile", "email", "offline_access"]` | Present |
| `claims_supported` | `["sub", "email", "name", "picture", "groups", "preferred_username", "updated_at"]` | Present |
| `token_endpoint_auth_methods_supported` | `["client_secret_basic", "client_secret_post", "none", "tls_client_auth", "self_signed_tls_client_auth"]` | Present |
| `code_challenge_methods_supported` | `["S256", "plain"]` | Present (should remove `plain`) |
| `check_session_iframe` | `{issuer}/oauth/check_session` | Present |
| `backchannel_logout_supported` | `true` | Present |
| `end_session_endpoint` | `{issuer}/oauth/logout` | Present |

**JWKS endpoint** (`server.go:151`):
- Served at `/oauth/jwks`
- Returns RSA public key with `kty`, `use`, `alg`, `kid`, `n`, `e`
- Key is RSA only (safe)
- Key ID from `KeyProvider.KeyID()`

**Gateway JWKS client** (`middleware/middleware.go:343-465`):
- `JWKSClient` struct with `refreshJWKS()`, `GetKey()`, `StartRefresh()`
- Filters non-RSA keys (`kty != "RSA"` or `use != "sig"` are skipped)
- Background refresh goroutine with configurable interval
- Falls back to static public key file if JWKS fetch fails
- 10-second HTTP timeout

#### What Is Missing

| Gap | Severity | Description |
|-----|----------|-------------|
| No HTTPS enforcement on discovery | Medium | `jwks_uri` and endpoints can be `http://` if issuer is misconfigured |
| No Cache-Control on discovery | Medium | Discovery response has no caching headers; CDN may cache indefinitely or not at all |
| No Cache-Control on JWKS | Low | JWKS response has `Cache-Control: no-store` on a different endpoint (token), not on JWKS itself |
| No query param rejection | Medium | Discovery handler accepts query params — vulnerable to CDN cache poisoning |
| No per-tenant discovery | High | No `/.well-known/openid-configuration/{tenant_id}` pattern; single discovery for all tenants |
| No metadata signing | Low | No RFC 8414 `signed_metadata` field; relies on TLS alone |
| No algorithm validation on JWKS | Medium | Gateway `refreshJWKS()` skips non-RSA keys but doesn't validate minimum key size |
| No rate limiting on JWKS refresh | Medium | `StartRefresh` runs on a fixed interval but `GetKey()` falls back to static key without rate-limiting unknown kid refreshes |
| No metadata change monitoring | Low | No hash comparison or alerting on metadata changes |
| No ECDSA key support | Low | JWKS only exposes RSA keys; ES256 not supported |
| `plain` PKCE method advertised | Medium | `code_challenge_methods_supported` includes `plain` which is insecure (RFC 7636 deprecates it) |

#### Gateway JWKS Security Assessment

The gateway's `JWKSClient.refreshJWKS()` (middleware.go:399-444) has several security-relevant
behaviors:

- **Positive**: Filters out non-RSA keys and non-signing keys.
- **Positive**: Falls back to static key on fetch failure (fail-safe).
- **Positive**: Atomic key replacement under mutex lock.
- **Negative**: No minimum key size validation.
- **Negative**: No HTTPS enforcement on `jwksURL` (will happily fetch from `http://`).
- **Negative**: No response size limit (no `io.LimitReader`).
- **Negative**: No rate limiting on refresh triggered by unknown kid.

---

## 10. Gap Analysis & Recommendations

### Action Items

| # | Action | Effort | Priority | Impact |
|---|--------|--------|----------|--------|
| 1 | **Reject query params on discovery endpoint** — Add `RawQuery != ""` check in discovery handler to prevent CDN cache poisoning | 1 hour | P1 | Eliminates cache poisoning attack vector |
| 2 | **Add HTTPS enforcement** — Validate `issuer` URL scheme at startup; reject `http://` issuer configuration. Add `http.ErrAbortHandler` if TLS is not configured | 2 hours | P1 | Prevents MITM metadata tampering |
| 3 | **Implement per-tenant discovery** — Add `/.well-known/openid-configuration/{tenant_id}` handler that returns tenant-specific metadata and JWKS URI. Update `OIDCDiscoveryConfig` to include tenant context | 1-2 days | P2 | Enables per-tenant key isolation and scoped discovery |
| 4 | **Add Cache-Control and Vary headers** — Set `Cache-Control: public, max-age=3600, must-revalidate` on discovery and `Cache-Control: public, max-age=300` on JWKS. Add `Vary: Accept` | 1 hour | P2 | Prevents stale metadata and content-type confusion |
| 5 | **Remove `plain` PKCE method** — Remove `"plain"` from `CodeChallengeMethodsSupported` in discovery metadata. RFC 7636 Section 7.2 recommends `S256` only | 30 minutes | P2 | Eliminates PKCE downgrade attack vector |

### Bonus Enhancements (Lower Priority)

| # | Action | Effort | Priority |
|---|--------|--------|----------|
| 6 | Add minimum RSA key size validation in gateway `refreshJWKS()` | 2 hours | P3 |
| 7 | Add `io.LimitReader` to JWKS fetch (max 1MB response) | 30 minutes | P3 |
| 8 | Implement metadata hash comparison and change alerting | 4 hours | P3 |
| 9 | Add ECDSA key support to JWKS and discovery metadata | 1 day | P4 |
| 10 | Implement OIDC Federation (RFC 8414 signed metadata) | 3-5 days | P4 |

### Summary

The GGID OIDC discovery implementation covers the core OIDC Discovery 1.0 specification with
all required fields and a safe default algorithm (RS256 only). The primary gaps are:

1. **CDN cache poisoning prevention** — trivial to fix (reject query params).
2. **Multi-tenant isolation** — significant architectural change but critical for production
   multi-tenant deployment.
3. **PKCE hardening** — remove `plain` method from advertised capabilities.
4. **JWKS transport security** — enforce HTTPS on JWKS URI and add response size limits.

These improvements align GGID with current OIDC security best practices from RFC 8414, RFC
7636 (PKCE), and the OWASP OAuth 2.0 Security Cheat Sheet.

---

## References

- [RFC 8414](https://datatracker.ietf.org/doc/html/rfc8414) — OAuth 2.0 Authorization Server Metadata
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517) — JSON Web Key (JWK)
- [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636) — Proof Key for Code Exchange (PKCE)
- [RFC 8444](https://datatracker.ietf.org/doc/html/rfc8444) — OAuth 2.0 Token Introspection
- [NIST SP 800-57](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final) — Key Management Recommendations
- [OWASP OAuth 2.0 Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html)
- [OIDC Federation 1.0](https://openid.net/specs/openid-federation-1_0.html) — Trust frameworks and signed metadata
