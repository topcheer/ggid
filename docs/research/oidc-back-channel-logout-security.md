# OIDC Back-Channel Logout Security Validation for IAM Systems

> Security-focused analysis of back-channel logout vulnerabilities, attack vectors,
> and validation hardening for multi-tenant IAM systems.
> Companion to `openid-connect-logout.md` (implementation patterns) — this document
> focuses exclusively on **security validation** and **attack surface**.
> Date: 2025-07-11 · Status: Research

---

## Table of Contents

1. [Back-Channel Logout Attack Surface](#1-back-channel-logout-attack-surface)
2. [Logout Token Validation Vulnerabilities](#2-logout-token-validation-vulnerabilities)
3. [Session Mapping for Logout](#3-session-mapping-for-logout)
4. [Logout Endpoint SSRF](#4-logout-endpoint-ssrf)
5. [Retry and Timeout Abuse](#5-retry-and-timeout-abuse)
6. [Cross-Tenant Logout Leakage](#6-cross-tenant-logout-leakage)
7. [Logout Token Replay](#7-logout-token-replay)
8. [Logout Delivery Monitoring](#8-logout-delivery-monitoring)
9. [GGID Logout Security Audit](#9-ggid-logout-security-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Back-Channel Logout Attack Surface

OIDC Back-Channel Logout (RFC 8417) is fundamentally different from front-channel
logout because it operates over **server-to-server** HTTP with no user agent
present. The Identity Provider (IdP) sends a `POST` request containing a
`logout_token` JWT to each Relying Party's (RP's) registered
`backchannel_logout_uri`. This creates a unique threat landscape.

### 1.1 Why Back-Channel Logout Is Security-Critical

Unlike front-channel logout (which rides the user's browser session and inherits
browser security boundaries), back-channel logout:

- **Has no user context**: The request is a machine-to-machine HTTP call. There
  is no SameSite cookie, no Origin header, no user agent to validate.
- **Contains a JWT that grants session destruction authority**: Anyone who can
  forge or replay the `logout_token` can force-terminate arbitrary user sessions.
- **Reaches arbitrary URLs**: The IdP fetches `backchannel_logout_uri` values
  registered by clients, turning the IdP into an SSRF launchpad if those URIs
  are not validated.
- **Is asynchronous**: The IdP may queue or retry delivery. Slow or malicious
  RP endpoints can cause resource exhaustion (DoS).

### 1.2 Primary Attack Vectors

| Attack | Precondition | Impact |
|--------|-------------|--------|
| **Forged logout token** | Token not signature-verified | Attacker terminates any user's session at will |
| **Token replay** | Captured token + no `jti` tracking | Repeated session termination, denial of service |
| **SSRF via logout URI** | Attacker controls client registration | IdP fetches internal URLs (cloud metadata, internal services) |
| **Algorithm confusion** | Token uses `alg: none` or HS256 with RSA public key | Attacker forges valid-looking token without private key |
| **Timing/slow-loris DoS** | RP endpoint accepts connection but never responds | IdP exhausts connection pool, cannot process logouts |
| **Cross-tenant leakage** | Tenant isolation not enforced in logout delivery | Tenant A logout triggers Tenant B session destruction |

### 1.3 The Trust Boundary Problem

The fundamental security question is: **how does the RP know the logout token
is genuinely from the IdP?** The answer is JWT signature verification against
the IdP's JWKS. However, many implementations (including GGID's current code)
parse logout tokens without verifying the signature, treating them as
informational rather than authoritative. This is the single most dangerous
security gap.

---

## 2. Logout Token Validation Vulnerabilities

RFC 8417 Section 2.4 defines strict requirements for logout tokens. Failure to
enforce every check creates an exploitable vulnerability.

### 2.1 Required Claims and Common Mistakes

| Claim | Requirement | Common Mistake |
|-------|------------|----------------|
| `iss` | MUST match IdP issuer | Not validated — attacker uses arbitrary issuer |
| `aud` | MUST contain the RP's client_id | Not validated — token intended for RP-A replayed at RP-B |
| `iat` | MUST be present | Missing — no way to detect stale/replayed tokens |
| `jti` | MUST be unique per token | Not tracked — infinite replay |
| `sub` | Optional, but if present MUST match known subject | Not correlated — attacker uses fake `sub` |
| `sid` | Optional, but if present MUST match known session | Not correlated — attacker uses fake `sid` |
| `events` | MUST contain `http://schemas.openid.net/event/backchannel-logout` | Not checked — any JWT is accepted as logout |
| `nonce` | MUST NOT be present | Not rejected — confusion with ID token |

### 2.2 Algorithm Confusion Attack

An attacker crafts a logout token with `alg: none` or `alg: HS256`. If the RP
uses a generic JWT parser that accepts any algorithm, the token appears valid.

```go
// VULNERABLE: accepts any algorithm
token, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
    return publicKey, nil // no algorithm check
})
```

### 2.3 Secure Logout Token Validation (Go)

The following code enforces all RFC 8417 requirements:

```go
package main

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LogoutTokenValidator validates OIDC back-channel logout tokens per RFC 8417.
type LogoutTokenValidator struct {
	issuer     string
	clientID   string
	publicKey  *rsa.PublicKey
	jtiChecker *JTIChecker // replay prevention store
}

// NewLogoutTokenValidator creates a validator bound to a specific issuer and client.
func NewLogoutTokenValidator(issuer, clientID string, pub *rsa.PublicKey, jti *JTIChecker) *LogoutTokenValidator {
	return &LogoutTokenValidator{
		issuer:     issuer,
		clientID:   clientID,
		publicKey:  pub,
		jtiChecker: jti,
	}
}

// ValidateLogoutToken performs all RFC 8417 security checks.
func (v *LogoutTokenValidator) ValidateLogoutToken(tokenStr string) (jwt.MapClaims, error) {
	// Step 1: Parse and verify signature — enforce RS256 only.
	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Reject any algorithm other than RS256.
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Step 2: Validate issuer.
	iss, _ := claims["iss"].(string)
	if iss != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, iss)
	}

	// Step 3: Validate audience — must contain our client_id.
	audValid := false
	switch aud := claims["aud"].(type) {
	case string:
		audValid = aud == v.clientID
	case []any:
		for _, a := range aud {
			if s, ok := a.(string); ok && s == v.clientID {
				audValid = true
				break
			}
		}
	}
	if !audValid {
		return nil, fmt.Errorf("audience does not contain client_id %s", v.clientID)
	}

	// Step 4: Validate iat is present and not in the future.
	iatRaw, hasIAT := claims["iat"]
	if !hasIAT {
		return nil, fmt.Errorf("missing 'iat' claim")
	}
	iat, err := claims.GetIssuedAt()
	if err != nil || iat == nil {
		return nil, fmt.Errorf("invalid 'iat' claim: %v", iatRaw)
	}
	if iat.After(time.Now().Add(30 * time.Second)) {
		return nil, fmt.Errorf("token issued in the future")
	}

	// Step 5: Require sub OR sid (at least one must be present).
	sub, hasSub := claims["sub"].(string)
	sid, hasSid := claims["sid"].(string)
	if !hasSub && !hasSid && sub == "" && sid == "" {
		return nil, fmt.Errorf("logout token must contain 'sub' or 'sid'")
	}

	// Step 6: Require events claim with backchannel-logout event.
	events, ok := claims["events"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing 'events' claim")
	}
	if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return nil, fmt.Errorf("events does not contain backchannel-logout event")
	}

	// Step 7: Reject nonce (per spec, logout tokens must not contain nonce).
	if _, ok := claims["nonce"]; ok {
		return nil, fmt.Errorf("logout token must not contain 'nonce'")
	}

	// Step 8: Enforce jti single-use for replay prevention.
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return nil, fmt.Errorf("missing 'jti' claim — required for replay prevention")
	}
	if v.jtiChecker != nil {
		if !v.jtiChecker.CheckAndStore(jti) {
			return nil, fmt.Errorf("replay detected: jti %s already used", jti)
		}
	}

	return claims, nil
}

// JTIChecker tracks seen JTIs with TTL to prevent token replay.
type JTIChecker struct {
	store    map[string]time.Time
	ttl      time.Duration
}

func NewJTIChecker(ttl time.Duration) *JTIChecker {
	return &JTIChecker{store: make(map[string]time.Time), ttl: ttl}
}

// CheckAndStore returns true if the jti is new, false if already seen.
func (c *JTIChecker) CheckAndStore(jti string) bool {
	// Purge expired entries first.
	now := time.Now()
	for k, t := range c.store {
		if now.Sub(t) > c.ttl {
			delete(c.store, k)
		}
	}
	if _, exists := c.store[jti]; exists {
		return false
	}
	c.store[jti] = now
	return true
}
```

### 2.4 What Can Go Wrong Without These Checks

- **Missing `events` check**: Any JWT (including access tokens or ID tokens)
  can be submitted as a logout token. An attacker steals a user's access token
  and POSTs it to the RP's backchannel endpoint, triggering logout.
- **Missing `iss`/`aud` check**: A logout token from a different IdP or intended
  for a different RP is accepted. In federated environments, this enables
  cross-provider logout attacks.
- **Missing `jti` enforcement**: A captured token can be replayed indefinitely,
  preventing the user from maintaining a session (persistent DoS).

---

## 3. Session Mapping for Logout

When the IdP decides to log out a user, it must determine **which RPs to notify**.
This requires mapping from the user's session to the RPs that participated in it.

### 3.1 Why `sub` Alone Is Insufficient

A user (identified by `sub`) may have multiple concurrent sessions: one from
their laptop, one from their phone, one from a shared kiosk. If the IdP only
sends `sub` in the logout token, the RP must terminate **all** sessions for that
user — including the one on the phone when the user only logged out on the laptop.

The `sid` (session ID) claim solves this: each OP-initiated session gets a unique
`sid`, and the logout token carries the specific `sid` to terminate. The RP
only kills the session matching that `sid`, preserving other concurrent sessions.

### 3.2 Session Table Design (PostgreSQL)

```sql
-- Sessions table: maps each authenticated session to user, tenant, and RPs.
CREATE TABLE oidc_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sid             VARCHAR(128) UNIQUE NOT NULL,   -- OIDC session identifier
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked         BOOLEAN NOT NULL DEFAULT false
);

-- Per-RP session participation: tracks which RPs were involved in each session.
CREATE TABLE oidc_session_clients (
    session_id      UUID NOT NULL REFERENCES oidc_sessions(id) ON DELETE CASCADE,
    client_id       VARCHAR(128) NOT NULL,
    backchannel_uri TEXT,                            -- RP's registered logout endpoint
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, client_id)
);

CREATE INDEX idx_sessions_user_tenant ON oidc_sessions(tenant_id, user_id) WHERE revoked = false;
CREATE INDEX idx_session_clients_uri ON oidc_session_clients(client_id);
```

### 3.3 Session-to-RP Mapper (Go)

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionMapper maps sessions to participating RPs for logout delivery.
type SessionMapper struct {
	db *sql.DB
}

// RPLogoutTarget describes an RP that needs back-channel logout notification.
type RPLogoutTarget struct {
	ClientID       string
	BackchannelURI string
}

// GetLogoutTargets returns all RPs that participated in a session.
// Used when the IdP initiates back-channel logout for a specific session.
func (m *SessionMapper) GetLogoutTargets(ctx context.Context, sid string) ([]RPLogoutTarget, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT sc.client_id, sc.backchannel_uri
		FROM oidc_session_clients sc
		JOIN oidc_sessions s ON s.id = sc.session_id
		WHERE s.sid = $1 AND s.revoked = false AND s.expires_at > now()
	`, sid)
	if err != nil {
		return nil, fmt.Errorf("query session clients: %w", err)
	}
	defer rows.Close()

	var targets []RPLogoutTarget
	for rows.Next() {
		var t RPLogoutTarget
		if err := rows.Scan(&t.ClientID, &t.BackchannelURI); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// GetLogoutTargetsByUser returns all active RPs for a user (for global logout).
func (m *SessionMapper) GetLogoutTargetsByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]RPLogoutTarget, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT DISTINCT sc.client_id, sc.backchannel_uri
		FROM oidc_session_clients sc
		JOIN oidc_sessions s ON s.id = sc.session_id
		WHERE s.tenant_id = $1 AND s.user_id = $2
		  AND s.revoked = false AND s.expires_at > now()
	`, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("query user sessions: %w", err)
	}
	defer rows.Close()

	var targets []RPLogoutTarget
	for rows.Next() {
		var t RPLogoutTarget
		if err := rows.Scan(&t.ClientID, &t.BackchannelURI); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// RevokeSession marks a session as revoked and returns affected RPs for notification.
func (m *SessionMapper) RevokeSession(ctx context.Context, sid string) ([]RPLogoutTarget, error) {
	targets, err := m.GetLogoutTargets(ctx, sid)
	if err != nil {
		return nil, err
	}

	_, err = m.db.ExecContext(ctx, `
		UPDATE oidc_sessions SET revoked = true WHERE sid = $1
	`, sid)
	if err != nil {
		return nil, fmt.Errorf("revoke session: %w", err)
	}

	return targets, nil
}
```

### 3.4 Session ID Generation

The `sid` must be cryptographically random and unpredictable. If an attacker
can guess or enumerate `sid` values, they can forge logout tokens that target
specific sessions.

```go
// GenerateSID creates a cryptographically random session ID for OIDC.
func GenerateSID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate sid: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

---

## 4. Logout Endpoint SSRF

### 4.1 The SSRF Vector

During back-channel logout, the IdP sends an HTTP POST to each RP's registered
`backchannel_logout_uri`. If an attacker can register a malicious client with
an internal URL (e.g., `http://169.254.169.254/latest/meta-data/` or
`http://localhost:8080/admin/delete-all`), the IdP will fetch that URL, acting
as an SSRF proxy.

This is especially dangerous because:

- The IdP runs inside the trusted network and can reach internal services.
- The POST body contains a JWT — an attacker can exfiltrate response data via
  timing or error messages.
- Cloud metadata endpoints (AWS IMDSv1, GCP) are accessible from compute
  instances.

### 4.2 Attack Scenario

1. Attacker registers an OAuth client via dynamic client registration (RFC 7591).
2. Attacker sets `backchannel_logout_uri` to `http://169.254.169.254/latest/meta-data/iam/security-credentials/`.
3. A user authenticates at the malicious client (establishing a session).
4. When the user logs out, the IdP POSTs to the SSRF URL, fetching cloud
   credentials in the response body.
5. Attacker reads the response (if the RP endpoint returns it) or uses timing
   to probe internal services.

### 4.3 Logout URI Validation (Go)

```go
package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	// blockedCIDRs are IP ranges that must never be reachable via logout delivery.
	blockedCIDRs = []string{
		"127.0.0.0/8",     // loopback
		"10.0.0.0/8",      // private
		"172.16.0.0/12",   // private
		"192.168.0.0/16",  // private
		"169.254.0.0/16",  // link-local (cloud metadata)
		"::1/128",         // IPv6 loopback
		"fc00::/7",        // IPv6 unique local
		"fe80::/10",       // IPv6 link-local
	}
)

// ValidateLogoutURI checks that a backchannel_logout_uri is safe for the IdP to fetch.
func ValidateLogoutURI(rawURI string) error {
	u, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	// Must use HTTPS in production (allow HTTP for localhost dev only).
	if u.Scheme != "https" {
		if !(u.Scheme == "http" && isLocalhost(u.Hostname())) {
			return fmt.Errorf("backchannel_logout_uri must use HTTPS")
		}
	}

	// Must have a host.
	if u.Host == "" {
		return fmt.Errorf("backchannel_logout_uri must have a host")
	}

	// Resolve hostname and check against blocked CIDRs.
	hostname := u.Hostname()
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %w", hostname, err)
	}

	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("backchannel_logout_uri resolves to blocked IP %s", ip)
		}
	}

	return nil
}

func isBlockedIP(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
```

### 4.4 Defense-in-Depth: Egress Network Policy

URI validation at registration time is not sufficient alone — DNS rebinding
attacks can bypass it (the domain resolves to a public IP during validation but
to an internal IP during the actual fetch). Production deployments should also:

- Apply egress firewall rules that block traffic to RFC 1918 and link-local
  ranges from the IdP's logout delivery worker.
- Pin the resolved IP at validation time and use a custom `http.Dialer` that
  rejects connections to IPs that differ from the pinned set.
- Run the logout delivery worker in a network namespace with restricted egress.

---

## 5. Retry and Timeout Abuse

### 5.1 The Slow-RP DoS Vector

Back-channel logout delivery is fire-and-forget from the user's perspective —
the user clicks logout and expects immediate feedback. But the IdP must notify
all participating RPs. If an RP's `backchannel_logout_uri` is slow (either
intentionally malicious or due to operational issues), the IdP's logout worker
can become blocked.

Attack scenario:

1. Attacker registers 100 clients, each with a `backchannel_logout_uri` pointing
   to a server that accepts connections but never responds.
2. A user authenticates at all 100 clients (or the attacker triggers logins).
3. When the user logs out, the IdP tries to POST to all 100 endpoints.
4. Each request hangs until the TCP timeout (default 30s+).
5. The IdP's logout worker pool is exhausted, and no further logouts can be
   processed.

### 5.2 Resilient Logout Delivery (Go)

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LogoutDeliverer sends logout tokens to RPs with timeout, retry, and circuit breaking.
type LogoutDeliverer struct {
	client         *http.Client
	maxRetries     int
	retryDelay     time.Duration
	circuitBreaker *CircuitBreaker
}

// NewLogoutDeliverer creates a deliverer with aggressive timeouts.
func NewLogoutDeliverer() *LogoutDeliverer {
	return &LogoutDeliverer{
		client: &http.Client{
			Timeout: 5 * time.Second, // hard timeout per request
			// Prevent redirect-based SSRF bypass.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxRetries:     3,
		retryDelay:     2 * time.Second,
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

// DeliverLogoutToken sends the logout token to a single RP.
// Returns nil on success (2xx response) or error after all retries.
func (d *LogoutDeliverer) DeliverLogoutToken(ctx context.Context, rpURI, token string) error {
	if d.circuitBreaker.IsOpen(rpURI) {
		return fmt.Errorf("circuit breaker open for %s", rpURI)
	}

	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d.retryDelay * time.Duration(1<<attempt)): // exponential backoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", rpURI,
			bytes.NewBufferString("logout_token="+token))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			d.circuitBreaker.RecordFailure(rpURI)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			d.circuitBreaker.RecordSuccess(rpURI)
			return nil
		}

		// 4xx = don't retry (RP rejected the token).
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			d.circuitBreaker.RecordSuccess(rpURI)
			return fmt.Errorf("RP rejected logout token: HTTP %d", resp.StatusCode)
		}

		// 5xx = retry.
		lastErr = fmt.Errorf("RP returned HTTP %d", resp.StatusCode)
		d.circuitBreaker.RecordFailure(rpURI)
	}

	return fmt.Errorf("logout delivery failed after %d retries: %w", d.maxRetries, lastErr)
}

// DeliverToAll sends logout tokens to all target RPs concurrently.
// Each RP is isolated — one failure doesn't block others.
func (d *LogoutDeliverer) DeliverToAll(ctx context.Context, targets []RPLogoutTarget, token string) []DeliveryResult {
	results := make([]DeliveryResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target RPLogoutTarget) {
			defer wg.Done()
			// Each RP gets its own context with timeout.
			rpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			err := d.DeliverLogoutToken(rpCtx, target.BackchannelURI, token)
			results[idx] = DeliveryResult{
				ClientID: target.ClientID,
				Success:  err == nil,
				Error:    err,
			}
		}(i, t)
	}

	wg.Wait()
	return results
}

type DeliveryResult struct {
	ClientID string
	Success  bool
	Error    error
}
```

### 5.3 Circuit Breaker per RP

```go
// CircuitBreaker implements per-endpoint circuit breaking to prevent
// repeated failures from exhausting resources.
type CircuitBreaker struct {
	mu           sync.Mutex
	failureCount map[string]int
	threshold    int
	cooldown     time.Duration
	openUntil    map[string]time.Time
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureCount: make(map[string]int),
		openUntil:    make(map[string]time.Time),
		threshold:    threshold,
		cooldown:     cooldown,
	}
}

func (cb *CircuitBreaker) IsOpen(key string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	until, ok := cb.openUntil[key]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(cb.openUntil, key)
		delete(cb.failureCount, key)
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordFailure(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount[key]++
	if cb.failureCount[key] >= cb.threshold {
		cb.openUntil[key] = time.Now().Add(cb.cooldown)
	}
}

func (cb *CircuitBreaker) RecordSuccess(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	delete(cb.failureCount, key)
	delete(cb.openUntil, key)
}
```

---

## 6. Cross-Tenant Logout Leakage

### 6.1 The Multi-Tenant Logout Problem

In a multi-tenant IAM system (like GGID), multiple organizations share the same
infrastructure. Each tenant has its own set of users, clients, and sessions.
A logout event for Tenant A must **never** affect Tenant B's sessions.

If the IdP uses `sub` (user ID) as the sole logout key, and user IDs are unique
per tenant, this is naturally safe. However, if `sid` is globally unique but not
tenant-scoped, or if the logout delivery worker does not filter targets by
tenant, cross-tenant leakage becomes possible.

### 6.2 Attack Scenario

1. Tenant A and Tenant B both use the IdP.
2. A user exists in both tenants (same person, different accounts).
3. If the IdP's session store does not include `tenant_id`, a logout for
   Tenant A's user may match Tenant B's session by `sub` collision.
4. More subtly: the session mapper queries for all sessions matching a `sub`
   without a `tenant_id` filter, returning sessions across tenants.

### 6.3 Tenant-Safe Logout Delivery (Go)

```go
package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// TenantSafeLogoutDeliverer ensures logout tokens are only delivered within the
// same tenant boundary.
type TenantSafeLogoutDeliverer struct {
	sessionMapper *SessionMapper
	deliverer     *LogoutDeliverer
	tokenIssuer   *LogoutTokenIssuer
}

// LogoutBySession performs tenant-scoped logout for a specific session.
func (t *TenantSafeLogoutDeliverer) LogoutBySession(
	ctx context.Context,
	tenantID uuid.UUID,
	sid string,
) ([]DeliveryResult, error) {
	// Verify the session belongs to this tenant before proceeding.
	belongsToTenant, err := t.sessionMapper.SessionBelongsToTenant(ctx, sid, tenantID)
	if err != nil {
		return nil, fmt.Errorf("verify tenant ownership: %w", err)
	}
	if !belongsToTenant {
		return nil, fmt.Errorf("session %s does not belong to tenant %s", sid, tenantID)
	}

	// Get logout targets — scoped to this session only.
	targets, err := t.sessionMapper.GetLogoutTargets(ctx, sid)
	if err != nil {
		return nil, fmt.Errorf("get logout targets: %w", err)
	}

	// Issue tenant-scoped logout token with tenant_id claim.
	token, err := t.tokenIssuer.IssueTenantScoped(ctx, tenantID, sid)
	if err != nil {
		return nil, fmt.Errorf("issue logout token: %w", err)
	}

	// Deliver to all RPs in this session.
	return t.deliverer.DeliverToAll(ctx, targets, token), nil
}

// LogoutByUser performs tenant-scoped global logout for a user.
func (t *TenantSafeLogoutDeliverer) LogoutByUser(
	ctx context.Context,
	tenantID, userID uuid.UUID,
) ([]DeliveryResult, error) {
	// Query sessions with explicit tenant_id filter.
	targets, err := t.sessionMapper.GetLogoutTargetsByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("get user logout targets: %w", err)
	}

	// Issue tenant-scoped logout token.
	token, err := t.tokenIssuer.IssueTenantScopedByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("issue logout token: %w", err)
	}

	return t.deliverer.DeliverToAll(ctx, targets, token), nil
}
```

### 6.4 SessionBelongsToTenant (SessionMapper addition)

```go
// SessionBelongsToTenant verifies that a session belongs to the given tenant.
func (m *SessionMapper) SessionBelongsToTenant(ctx context.Context, sid string, tenantID uuid.UUID) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM oidc_sessions
		WHERE sid = $1 AND tenant_id = $2 AND revoked = false
	`, sid, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

### 6.5 Tenant-Scoped `sid` Convention

For defense-in-depth, `sid` values should be prefixed with `tenant_id`:

```
sid = base64({tenant_id}:{random_bytes})
```

This ensures that even if a logout token's `sid` leaks across tenant boundaries,
the RP can verify the tenant prefix matches its own tenant, rejecting mismatched
tokens independently of the IdP.

---

## 7. Logout Token Replay

### 7.1 Replay Attack Description

If an attacker captures a logout token (e.g., from HTTP logs, a compromised RP,
or a network MITM if TLS is not enforced), they can replay it to:

- Force repeated session termination (DoS — user cannot stay logged in).
- Trigger RP processing of stale logout events.
- Exhaust RP resources if logout processing is expensive.

### 7.2 `jti` Tracking with TTL

The `jti` (JWT ID) claim makes each logout token unique. The RP must track
seen `jti` values and reject duplicates. The tracking store needs a TTL because
logout tokens have a natural lifetime — once the session they reference is
destroyed, replaying the token is a no-op but should still be rejected.

```go
// RedisJTIStore provides distributed, TTL-based jti tracking for replay prevention.
type RedisJTIStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisJTIStore(rdb *redis.Client, ttl time.Duration) *RedisJTIStore {
	return &RedisJTIStore{rdb: rdb, ttl: ttl}
}

// CheckAndStore atomically checks if a jti is new and stores it.
// Returns true if the jti was not seen before, false if duplicate.
func (s *RedisJTIStore) CheckAndStore(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("ggid:logout_jti:%s", jti)
	// SETNX = "set if not exists" — atomic check-and-set.
	result, err := s.rdb.SetNX(ctx, key, time.Now().Unix(), s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("jti store: %w", err)
	}
	return result, nil
}
```

### 7.3 Idempotent Logout Processing

Even with `jti` tracking, RP logout processing should be idempotent — processing
the same logout token twice should produce the same result (session destroyed)
without errors or side effects.

```go
// IdempotentLogoutHandler processes logout tokens idempotently.
type IdempotentLogoutHandler struct {
	validator  *LogoutTokenValidator
	sessionMgr *SessionManager
	jtiStore   *RedisJTIStore
}

func (h *IdempotentLogoutHandler) HandleLogout(ctx context.Context, tokenStr string) error {
	// Validate the token (including jti uniqueness check).
	claims, err := h.validator.ValidateLogoutToken(tokenStr)
	if err != nil {
		return err
	}

	jti, _ := claims["jti"].(string)

	// Double-check jti in Redis for distributed safety.
	isNew, err := h.jtiStore.CheckAndStore(ctx, jti)
	if err != nil {
		// If Redis is down, fail safe — don't process without replay protection.
		return fmt.Errorf("cannot verify jti: %w", err)
	}
	if !isNew {
		// Already processed — return success (idempotent).
		return nil
	}

	// Perform the actual session destruction.
	sid, _ := claims["sid"].(string)
	sub, _ := claims["sub"].(string)

	if sid != "" {
		return h.sessionMgr.DestroySession(ctx, sid)
	}
	if sub != "" {
		return h.sessionMgr.DestroyUserSessions(ctx, sub)
	}

	return fmt.Errorf("no valid session identifier in logout token")
}
```

### 7.4 TTL Selection

The `jti` store TTL should be at least as long as the maximum token lifetime
plus clock skew tolerance. For logout tokens, a 24-hour TTL is conservative:

- Logout tokens are typically short-lived (minutes).
- A 24-hour window catches replays within the same operational day.
- After 24 hours, the referenced session has certainly expired naturally.

---

## 8. Logout Delivery Monitoring

### 8.1 Why Monitoring Matters

Back-channel logout is a security-critical operation — if it fails silently,
sessions that should have been terminated remain active, creating a security
gap. Monitoring must cover:

- **Delivery failures**: RP returned an error or timed out.
- **Delivery rate anomalies**: sudden spike in logouts may indicate an attack.
- **Stale sessions**: sessions that should have been destroyed but weren't.

### 8.2 Logout Delivery Monitor (Go)

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

// LogoutMonitor tracks delivery results and detects anomalies.
type LogoutMonitor struct {
	db          LogoutDeliveryLog
	alertThresh int           // max failures before alert
	alertWindow time.Duration // rolling window for failure count
}

// DeliveryRecord captures the outcome of a single logout delivery.
type DeliveryRecord struct {
	SessionID  string
	TenantID   uuid.UUID
	ClientID   string
	Success    bool
	StatusCode int
	Error      string
	Timestamp  time.Time
}

// LogDelivery records the result of a logout token delivery.
func (m *LogoutMonitor) LogDelivery(ctx context.Context, rec DeliveryRecord) error {
	if err := m.db.Insert(ctx, rec); err != nil {
		return err
	}

	// Check if this RP has exceeded the failure threshold.
	if !rec.Success {
		failureCount, err := m.db.CountRecentFailures(ctx, rec.ClientID, m.alertWindow)
		if err == nil && failureCount >= m.alertThresh {
			log.Printf("[ALERT] RP %s has %d logout delivery failures in %s",
				rec.ClientID, failureCount, m.alertWindow)
			// In production: send to PagerDuty, Slack, etc.
		}
	}

	return nil
}

// ReconcileSessions detects sessions that should have been terminated but weren't.
// This is a periodic background job that cross-references:
//   - Sessions marked as revoked in the IdP
//   - Sessions still active at RPs (via session check iframe or token introspection)
func (m *LogoutMonitor) ReconcileSessions(ctx context.Context) ([]StaleSession, error) {
	// Find sessions revoked more than 5 minutes ago where delivery failed.
	stale, err := m.db.FindFailedDeliveries(ctx, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	for _, s := range stale {
		log.Printf("[RECONCILE] session %s for tenant %s RP %s — logout delivery failed, session may be stale",
			s.SessionID, s.TenantID, s.ClientID)
	}

	return stale, nil
}

type StaleSession struct {
	SessionID string
	TenantID  uuid.UUID
	ClientID  string
	FailedAt  time.Time
}
```

### 8.3 Metrics

The following metrics should be exposed via Prometheus:

| Metric | Type | Labels |
|--------|------|--------|
| `oidc_logout_deliveries_total` | counter | `tenant_id`, `client_id`, `status` |
| `oidc_logout_delivery_duration_seconds` | histogram | `client_id` |
| `oidc_logout_delivery_failures_total` | counter | `tenant_id`, `client_id`, `error_type` |
| `oidc_logout_token_replays_blocked_total` | counter | `tenant_id` |
| `oidc_logout_circuit_breaker_open_total` | counter | `client_id` |
| `oidc_logout_stale_sessions` | gauge | `tenant_id` |

---

## 9. GGID Logout Security Audit

### 9.1 Current Implementation

**Endpoints** (from `services/oauth/internal/server/server.go`):

| Route | Purpose | Issues |
|-------|---------|--------|
| `POST /oauth/logout` | RP-initiated logout | Uses `ParseAccessToken` (not logout token validation), returns sub/sid in response |
| `POST /api/v1/oauth/backchannel-logout` | Back-channel logout receiver | Uses `ParseBackchannelLogoutToken` |

**Token validation** (from `oauth_service.go:1382-1425`):

```go
func (s *OAuthService) ParseBackchannelLogoutToken(tokenStr string) (jwt.MapClaims, error) {
    token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
    // ...
}
```

### 9.2 Security Findings

| # | Finding | Severity | Evidence |
|---|---------|----------|----------|
| 1 | **No signature verification** — uses `ParseUnverified` | CRITICAL | `oauth_service.go:1384` |
| 2 | **No `iss` validation** | HIGH | No issuer check in `ParseBackchannelLogoutToken` |
| 3 | **No `aud` validation** | HIGH | No audience check — cross-RP token reuse possible |
| 4 | **No `iat`/`exp` validation** | HIGH | No temporal validation |
| 5 | **`jti` tracking is in-memory** (`sync.Map`) with no TTL | MEDIUM | `oauth_service.go:1417-1421` — entries never expire, memory leak |
| 6 | **`jti` tracking is not distributed** | MEDIUM | `sync.Map` is per-process — multi-instance deployments can't share jti state |
| 7 | **No back-channel logout delivery to RPs** | HIGH | Comment at `oauth_service.go:1376-1377`: "In a full implementation, this would iterate all registered client back-channel logout URIs" |
| 8 | **No `backchannel_logout_uri` field on OAuthClient** | HIGH | `domain/models.go` — client model has no logout URI field |
| 9 | **No logout URI validation** (SSRF) | HIGH | No validation code exists |
| 10 | **No tenant isolation in logout** | HIGH | `BackchannelLogout(sub)` takes only `sub`, no `tenant_id` |
| 11 | **`/oauth/logout` uses wrong parser** | MEDIUM | Uses `ParseAccessToken` instead of `ParseBackchannelLogoutToken` |
| 12 | **Discovery advertises `backchannel_logout_supported: true`** | MEDIUM | `oauth_service.go:383` — misleading since delivery is not implemented |

### 9.3 What Exists (Positive)

- Logout token format checks: `sub` or `sid` required, `events` claim checked,
  `nonce` rejected, `jti` uniqueness checked.
- Discovery endpoint advertises `backchannel_logout_supported` and
  `end_session_endpoint`.
- RP-initiated logout endpoint validates `post_logout_redirect_uri` URL format.
- `BackchannelLogoutEndpoint` service-layer method exists for clean separation.

### 9.4 What Is Missing

- **Signature verification**: The most critical gap — any JWT can be submitted
  as a logout token.
- **Session store**: No `oidc_sessions` or `oidc_session_clients` tables. The
  IdP cannot map sessions to RPs.
- **Logout delivery**: No HTTP client code to POST logout tokens to RPs.
- **Retry/circuit breaker**: No resilience layer for logout delivery.
- **Monitoring/reconciliation**: No logging of delivery outcomes or stale
  session detection.
- **Tenant scoping**: `BackchannelLogout` function signature has no tenant ID.
- **Logout URI registration and validation**: Client model has no
  `backchannel_logout_uri` field.

---

## 10. Gap Analysis & Recommendations

### 10.1 Priority Matrix

| Priority | Action | Effort | Risk Reduction |
|----------|--------|--------|---------------|
| P0 | **Add JWT signature verification** to `ParseBackchannelLogoutToken` | S (2h) | Eliminates token forgery — the single most dangerous attack |
| P0 | **Add `iss` and `aud` validation** | S (1h) | Prevents cross-issuer and cross-RP token abuse |
| P1 | **Add `iat`/`exp` validation** and reject future `iat` | S (1h) | Prevents stale token replay and clock-based attacks |
| P1 | **Move `jti` tracking to Redis** with TTL | M (4h) | Distributed replay prevention, eliminates memory leak |
| P1 | **Add `tenant_id` to `BackchannelLogout`** function signature | S (2h) | Prevents cross-tenant session leakage |
| P2 | **Add `backchannel_logout_uri` to `OAuthClient` model** with SSRF validation | M (1d) | Enables actual logout delivery with SSRF protection |
| P2 | **Implement logout delivery worker** with timeout, retry, circuit breaker | L (2-3d) | Completes the back-channel logout feature |
| P2 | **Create session mapping tables** (`oidc_sessions`, `oidc_session_clients`) | M (1d) | Enables per-session and per-RP logout targeting |
| P3 | **Fix `/oauth/logout` to use `ParseBackchannelLogoutToken`** | S (1h) | Eliminates wrong-parser vulnerability |
| P3 | **Add Prometheus metrics** for logout delivery | S (3h) | Enables operational visibility and alerting |
| P3 | **Add logout delivery reconciliation job** | M (1d) | Detects and remediates stale sessions |

### 10.2 Recommended Implementation Order

**Phase 1 — Token Security (P0/P1, ~1 day):**
Fix the most critical validation gaps. These changes are localized to
`ParseBackchannelLogoutToken` and have no external dependencies.

**Phase 2 — Session Infrastructure (P2, ~3 days):**
Build the session tables, add `backchannel_logout_uri` to client model, and
wire up the delivery worker. This is the bulk of the implementation effort.

**Phase 3 — Monitoring & Hardening (P3, ~2 days):**
Add metrics, reconciliation, and tenant-scoped `sid` convention. These are
production-readiness improvements that should follow the core feature.

### 10.3 Testing Recommendations

- **Fuzz test** `ParseBackchannelLogoutToken` with malformed JWTs to verify
  rejection of all non-conforming inputs.
- **Integration test** the full logout flow: login at multiple RPs → trigger
  logout → verify all RP endpoints receive tokens and sessions are destroyed.
- **Chaos test**: simulate slow RP endpoints (50% response latency increase) and
  verify the circuit breaker triggers.
- **Security test**: attempt token replay, cross-tenant logout, and SSRF
  injection to verify all protections are effective.

---

## References

- **RFC 8417**: OpenID Connect Back-Channel Logout 1.0
- **RFC 7519**: JSON Web Token (JWT) — Section 4.1.7 (`jti` claim)
- **OIDC Session Management 1.0**: `check_session_iframe` specification
- **OIDC Front-Channel Logout 1.0**: Comparison and trade-offs vs. back-channel
- **OWASP ASVS 3.0.1**: Session Management verification requirements
- `openid-connect-logout.md`: GGID implementation patterns companion document
