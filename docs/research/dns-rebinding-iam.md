# DNS Rebinding Attacks on IAM Systems

> Research document for the GGID IAM Suite. Covers DNS rebinding mechanics, SSRF
> via metadata endpoints, host header validation, TOCTOU on DNS resolution, and
> concrete Go defense implementations. Includes source-level analysis of GGID's
> gateway, webhook delivery, and SAML components.

---

## Table of Contents

1. [DNS Rebinding Attack Mechanics](#1-dns-rebinding-attack-mechanics)
2. [SSRF via Metadata Endpoints](#2-ssrf-via-metadata-endpoints)
3. [Host Header Validation](#3-host-header-validation)
4. [TOCTOU on DNS Resolution](#4-toctou-on-dns-resolution)
5. [Defense: Host Header Allowlist Middleware](#5-defense-host-header-allowlist-middleware)
6. [Defense: DNS Pinning and Rebinding Protection](#6-defense-dns-pinning-and-rebinding-protection)
7. [Webhook URL Validation](#7-webhook-url-validation)
8. [GGID Gateway Host Header Analysis](#8-ggid-gateway-host-header-analysis)
9. [Gap Analysis and Recommendations](#9-gap-analysis-and-recommendations)

---

## 1. DNS Rebinding Attack Mechanics

### 1.1 What Is DNS Rebinding?

DNS rebinding is a technique that bypasses the same-origin policy (SOP) enforced
by web browsers. The attacker controls a DNS authoritative server for a domain
they own (e.g., `evil.example.com`). When a victim's browser resolves that
domain, the DNS server returns a short-TTL A record pointing to the attacker's
IP address. On a subsequent resolution (seconds later), the DNS server returns a
**different** IP address — typically the loopback address `127.0.0.1`, an RFC
1918 private IP (`10.0.0.x`, `192.168.x.x`, `172.16.x.x`), or the link-local
cloud metadata address `169.254.169.254`.

Because the browser checks the origin against the hostname (`evil.example.com`)
and not against the IP address, JavaScript loaded from the first resolution
continues to run under the `evil.example.com` origin. When that JavaScript makes
fetch/XHR requests to `http://evil.example.com/admin`, the browser resolves the
hostname again, gets the victim's internal IP, and sends the request to the
internal service — all while the browser's SOP believes it is a same-origin
request to `evil.example.com`.

### 1.2 Attack Flow

```
Step 1: Victim visits http://evil.example.com/
  DNS lookup → TTL=1s → A: 203.0.113.50 (attacker server)
  Browser loads malicious JS from 203.0.113.50
  Origin: http://evil.example.com

Step 2: Malicious JS waits 2 seconds, then:
  fetch("http://evil.example.com:8080/api/v1/admin/users")
  DNS lookup → TTL=1s → A: 127.0.0.1 (victim's local service)
  Browser sends request to 127.0.0.1:8080
  Origin header: http://evil.example.com (spoofed origin)
  Cookie header: none (cross-origin, no credentials)
  BUT: internal services often have NO auth on localhost
```

### 1.3 Why IAM Systems Are Prime Targets

IAM admin consoles and APIs are high-value DNS rebinding targets because:

- **Admin endpoints expose full user management** — create/delete users, assign
  roles, rotate API keys, modify policies. A successful rebinding attack yields
  complete identity infrastructure compromise.
- **Internal-only admin APIs** — many IAM deployments expose management
  endpoints on internal ports without authentication, relying on network
  isolation. DNS rebinding punches through that isolation.
- **Token issuance endpoints** — if an attacker can reach an internal OAuth
  token endpoint, they may obtain access tokens with no user credentials.
- **JWKS / signing key endpoints** — rebinding to an internal JWKS endpoint
  can leak cryptographic material.
- **Metadata services** — in cloud deployments, IAM services run on EC2/ECS/EKS.
  If the service makes outbound HTTP calls (SAML metadata fetch, OIDC discovery,
  webhook delivery), DNS rebinding can redirect those calls to
  `169.254.169.254` to steal instance credentials.

### 1.4 Real-World DNS Rebinding Attacks

| Year | Target | Impact |
|------|--------|--------|
| 2018 | **Ethereum cryptocurrency wallets** — geth/parity JSON-RPC on localhost | Attackers stole funds by rebinding to `localhost:8545` and sending transactions from unlocked wallets |
| 2018 | **Torrent clients** — uTorrent/Transmission web UI | Remote code execution via rebinding to localhost API |
| 2018 | **DVRs and IoT devices** | Full device takeover via unauthenticated localhost admin panels |
| 2021 | **HashiCorp Consul/Vault** | Internal API access bypassing network controls |
| 2022 | **Redis/Memcached on localhost** | Data exfiltration and cache poisoning |

The common thread: **services that trust localhost or internal IPs without
authentication are vulnerable to DNS rebinding from any web page the user
visits.**

---

## 2. SSRF via Metadata Endpoints

### 2.1 Cloud Metadata Services

Every major cloud provider exposes a metadata service reachable from within
virtual machines at `169.254.169.254` (link-local address):

- **AWS IMDSv1**: `http://169.254.169.254/latest/meta-data/iam/security-credentials/<role>/`
  returns temporary access keys for the instance's IAM role. No authentication
  required with IMDSv1.
- **GCP**: `http://metadata.google.internal/computeMetadata/v1/` returns service
  account tokens when `Metadata-Flavor: Google` header is present.
- **Azure**: `http://169.254.169.254/metadata/identity/oauth2/token` returns
  managed identity tokens.

### 2.2 IAM Services as SSRF Vectors

GGID and similar IAM systems make several types of outbound HTTP requests:

1. **SAML IdP metadata fetch** — an admin configures a SAML IdP metadata URL.
   The IAM service fetches XML metadata from that URL. If the URL points to a
   hostname controlled by the attacker, DNS rebinding redirects the fetch to
   `169.254.169.254`.

2. **OIDC discovery** — the IAM service fetches
   `https://<idp>/.well-known/openid-configuration` from the configured IdP.
   Same rebinding risk.

3. **Webhook delivery** — when a user event fires (registration, login,
   password change), the IAM service POSTs to a user-provided webhook URL.
   An attacker registers a webhook pointing to
   `http://169.254.169.254/latest/meta-data/` and receives cloud credentials
   in the webhook response (if the service reads responses).

4. **SCIM provisioning** — outbound SCIM API calls to user-provided endpoints.

5. **Social login API calls** — GGID's `pkg/social` package makes hardcoded
   calls to `api.github.com` and `graph.microsoft.com`. These are safe because
   the URLs are constants, not user-controlled. But any user-provided URL is a
   potential SSRF vector.

### 2.3 Attack Scenario Against GGID SAML Metadata Fetch

```
Attacker (tenant admin) configures SAML IdP:
  Metadata URL: http://my-idp.evil.example.com/metadata.xml

DNS server for evil.example.com:
  First lookup:  TTL=1s → 203.0.113.50 (attacker's metadata XML)
  Second lookup: TTL=1s → 169.254.169.254 (AWS metadata)

GGID service flow:
  1. Admin saves SAML config → triggers metadata fetch
  2. First DNS resolution → 203.0.113.50 → fetches XML, validates, saves IdP cert
  3. Background re-fetch (every 60 minutes) → DNS now resolves to 169.254.169.254
  4. Fetch GET http://169.254.169.254/latest/meta-data/iam/security-credentials/
  5. AWS returns temporary access keys in XML format
  6. Attacker uses these keys to access AWS resources as the IAM role
```

Even if the metadata fetch response is not directly returned to the attacker,
the attacker can use the SAML assertion processing path as an oracle, or exploit
the fact that the fetched "metadata" is parsed and stored — a carefully crafted
response from `169.254.169.254` could inject a malicious IdP certificate into
the SAML configuration.

---

## 3. Host Header Validation

### 3.1 Why Host Header Validation Stops DNS Rebinding

DNS rebinding relies on the browser believing that `evil.example.com` is the
origin for all requests. But the **Host header** in each HTTP request contains
the hostname the browser resolved. When the browser sends a request after
rebinding, the Host header is still `evil.example.com`, not `localhost` or the
internal IP.

If the server validates the Host header against a known allowlist of expected
hostnames (e.g., `iam.ggid.io`, `console.ggid.io`), any request with
`Host: evil.example.com` is rejected with `400 Bad Request` before it reaches
any application logic.

```
Request after rebinding:
  GET /api/v1/users HTTP/1.1
  Host: evil.example.com    ← rejected by allowlist middleware
  ...

Response:
  HTTP/1.1 400 Bad Request
  {"error": "unknown host"}
```

### 3.2 Allowlist Strategy

The allowlist should include:

- Production hostnames: `iam.ggid.io`, `api.ggid.io`, `console.ggid.io`
- Staging hostnames: `staging-iam.ggid.io`
- Loopback for development: `localhost`, `127.0.0.1`
- Internal service mesh names: `gateway.ggid.svc.cluster.local`
- Wildcard patterns for multi-tenant subdomains: `*.iam.ggid.io`

### 3.3 Edge Cases

**X-Forwarded-Host**: When behind a reverse proxy (NGINX, ALB), the original
Host header is preserved by the proxy, but some proxies set `X-Forwarded-Host`
instead. The middleware should validate `X-Forwarded-Host` if present (but only
when behind a trusted proxy), falling back to `r.Host`.

**Multiple Host headers**: RFC 7230 states that a server MUST respond with
`400 Bad Request` if a request contains multiple Host headers. Go's `net/http`
server already collapses duplicate headers into `r.Host`, but the raw header map
may still contain duplicates. The middleware should check
`r.Header["Host"]` (the raw slice) for length > 1.

**IPv6 addresses**: `[::1]`, `[fe80::1]` must be handled when stripping ports.
Use `net.SplitHostPort` which correctly handles IPv6 bracket notation.

**Empty Host**: A request with no Host header (HTTP/1.0) should be rejected
unless explicitly allowed for health check endpoints.

---

## 4. TOCTOU on DNS Resolution

### 4.1 Time-of-Check to Time-of-Use

Even when an application validates a URL's resolved IP address before connecting,
a race condition exists:

```
1. Application resolves hostname → gets IP 93.184.216.34 (safe, external)
2. Application validates IP → passes check (not in RFC 1918 range)
3. DNS cache expires (TTL=1)
4. Application dials the hostname → DNS resolves again → gets 169.254.169.254
5. Connection goes to cloud metadata service
```

This is the classic TOCTOU (time-of-check/time-of-use) vulnerability applied to
DNS resolution. The check at step 2 used one IP; the use at step 4 connected to
a different IP.

### 4.2 Why Pinning the Resolved IP Matters

The solution is to **pin** the resolved IP address for the duration of the
connection:

1. Resolve the hostname once.
2. Validate the resolved IP against the blocklist.
3. Dial the **validated IP address directly** (not the hostname) — bypassing
   the Go HTTP client's own DNS resolution.

This eliminates the window between check and use because the same IP address is
used for both validation and connection.

### 4.3 Go Code: IP Pinning with net.Resolver

```go
package ssrf

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// SafeDialer resolves the hostname, validates the IP, and dials the validated
// IP directly — preventing TOCTOU races on DNS resolution.
type SafeDialer struct {
	Resolver  *net.Resolver
	Timeout   time.Duration
	Blocklist *IPBlocklist
}

// DialContext implements the dial function used by http.Transport.
// It resolves the host, validates each candidate IP, and connects to the
// first allowed address.
func (d *SafeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host:port: %w", err)
	}

	// If the address is already an IP literal, validate directly.
	if ip := net.ParseIP(host); ip != nil {
		if d.Blocklist.IsBlocked(ip) {
			return nil, fmt.Errorf("blocked IP: %s", ip)
		}
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}

	// Resolve hostname to IP addresses.
	resolver := d.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup %s: %w", host, err)
	}

	// Filter out blocked IPs.
	for _, ipAddr := range ips {
		if d.Blocklist.IsBlocked(ipAddr.IP) {
			continue
		}
		// Dial the validated IP directly — this is the PIN.
		// We connect to the IP, not the hostname, so there is no second DNS
		// lookup that could resolve to a different address.
		var dialer net.Dialer
		dialer.Timeout = d.Timeout
		return dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
	}

	return nil, fmt.Errorf("all resolved IPs for %s are blocked", host)
}

// SafeTransport returns an http.Transport that uses SafeDialer, preventing
// connections to internal IP ranges.
func SafeTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&SafeDialer{
			Timeout:   10 * time.Second,
			Blocklist: DefaultBlocklist(),
		}).DialContext,
	}
}

// SafeHTTPClient returns an http.Client that cannot connect to internal IPs.
func SafeHTTPClient() *http.Client {
	return &http.Client{
		Transport: SafeTransport(),
		Timeout:   30 * time.Second,
	}
}
```

### 4.4 Why Go's Default Transport Is Vulnerable

Go's `http.DefaultTransport` uses `net.Dialer.DialContext`, which resolves the
hostname via the system resolver on each call. There is no IP validation step.
An attacker who controls DNS can redirect any outbound HTTP request to an
internal address at any time. The `SafeDialer` above fixes this by interposing
validation between resolution and connection.

---

## 5. Defense: Host Header Allowlist Middleware

### 5.1 Complete Go Middleware

```go
package middleware

import (
	"net"
	"net/http"
	"strings"
)

// HostAllowlistConfig configures the host header validation middleware.
type HostAllowlistConfig struct {
	// AllowedHosts is the explicit set of hostnames to accept.
	// Comparison is case-insensitive. Ports are stripped before comparison.
	// Example: []string{"iam.ggid.io", "api.ggid.io", "localhost"}
	AllowedHosts []string

	// AllowedSuffixes permits wildcard-style matching for multi-tenant
	// subdomains. Example: []string{".iam.ggid.io"} matches
	// "acme.iam.ggid.io" and "contoso.iam.ggid.io".
	AllowedSuffixes []string

	// TrustProxy, when true, checks X-Forwarded-Host before r.Host.
	// Only enable this when running behind a trusted reverse proxy that
	// overwrites client-supplied X-Forwarded-Host.
	TrustProxy bool

	// AllowEmptyHost permits requests with no Host header (HTTP/1.0).
	// Restrict to health check paths in production.
	AllowEmptyHost bool
}

// hostAllowlist is a case-insensitive map for O(1) lookup.
type hostAllowlist struct {
	exact   map[string]bool
	suffix  []string
	cfg     HostAllowlistConfig
}

func newHostAllowlist(cfg HostAllowlistConfig) *hostAllowlist {
	ha := &hostAllowlist{
		exact:  make(map[string]bool, len(cfg.AllowedHosts)),
		suffix: cfg.AllowedSuffixes,
		cfg:    cfg,
	}
	for _, h := range cfg.AllowedHosts {
		ha.exact[strings.ToLower(strings.TrimSpace(h))] = true
	}
	return ha
}

func (ha *hostAllowlist) isAllowed(host string) bool {
	if host == "" {
		return ha.cfg.AllowEmptyHost
	}
	// Strip port (handles IPv6 bracket notation via net.SplitHostPort).
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.ToLower(host)

	// Exact match.
	if ha.exact[host] {
		return true
	}

	// Suffix match for wildcard subdomains.
	for _, suf := range ha.suffix {
		if strings.HasSuffix(host, strings.ToLower(suf)) {
			return true
		}
	}
	return false
}

// HostAllowlistMiddleware returns middleware that rejects requests whose Host
// header does not match the configured allowlist. Rejected requests receive
// HTTP 400 with a JSON error body.
func HostAllowlistMiddleware(cfg HostAllowlistConfig) func(http.Handler) http.Handler {
	ha := newHostAllowlist(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Reject multiple Host headers (RFC 7230 §5.4).
			if rawHost := r.Header["Host"]; len(rawHost) > 1 {
				http.Error(w, `{"error":"multiple host headers"}`, http.StatusBadRequest)
				return
			}

			host := r.Host
			if ha.cfg.TrustProxy {
				if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
					host = xfh
				}
			}

			if !ha.isAllowed(host) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"unknown host"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

### 5.2 Integration with Reverse Proxy

```go
func main() {
	gateway := router.New(cfg, jwks)

	// Wrap the gateway with host allowlist as the outermost middleware.
	// This runs before JWT verification, CORS, and rate limiting.
	hostValidator := middleware.HostAllowlistMiddleware(middleware.HostAllowlistConfig{
		AllowedHosts: []string{
			"iam.ggid.io",
			"api.ggid.io",
			"localhost",
			"127.0.0.1",
		},
		AllowedSuffixes: []string{
			".iam.ggid.io",   // multi-tenant: acme.iam.ggid.io
		},
		TrustProxy:       true,  // behind ALB/NGINX
		AllowEmptyHost:   false,
	})

	chain := middleware.Chain(
		hostValidator,
		middleware.RequestID(),
		middleware.CORS(cfg.AllowedOrigins),
		middleware.RateLimit(cfg.RateLimit),
		middleware.JWTAuth(cfg.JWTSecret, publicPaths),
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: chain(gateway),
	}
	server.ListenAndServe()
}
```

### 5.3 Testing the Middleware

```go
func TestHostAllowlist_RejectsUnknown(t *testing.T) {
	mw := HostAllowlistMiddleware(HostAllowlistConfig{
		AllowedHosts: []string{"iam.ggid.io"},
	})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHostAllowlist_AcceptsSuffix(t *testing.T) {
	mw := HostAllowlistMiddleware(HostAllowlistConfig{
		AllowedSuffixes: []string{".iam.ggid.io"},
	})
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Host = "acme.iam.ggid.io:8080"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called for allowed suffix")
	}
}
```

---

## 6. Defense: DNS Pinning and Rebinding Protection

### 6.1 IP Blocklist

```go
package ssrf

import (
	"net"
)

// IPBlocklist determines whether an IP address should be blocked for outbound
// connections from external-facing services.
type IPBlocklist struct {
	privateNets []*net.IPNet
	loopback    *net.IPNet
	linkLocal   *net.IPNet
}

// DefaultBlocklist returns a blocklist covering all non-routable and internal
// IP ranges: RFC 1918 private, loopback, link-local (including 169.254.169.254),
// and unspecified addresses.
func DefaultBlocklist() *IPBlocklist {
	bl := &IPBlocklist{}

	// RFC 1918 private ranges.
	for _, cidr := range []string{
		"10.0.0.0/8",      // private class A
		"172.16.0.0/12",   // private class B
		"192.168.0.0/16",  // private class C
		"0.0.0.0/8",       // "this network"
		"100.64.0.0/10",   // CGNAT (RFC 6598)
		"169.254.0.0/16",  // link-local (cloud metadata!)
		"224.0.0.0/4",     // multicast
		"240.0.0.0/4",     // reserved
	} {
		_, n, _ := net.ParseCIDR(cidr)
		bl.privateNets = append(bl.privateNets, n)
	}

	// IPv6 ranges.
	for _, cidr := range []string{
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique-local
		"ff00::/8",       // IPv6 multicast
	} {
		_, n, _ := net.ParseCIDR(cidr)
		bl.privateNets = append(bl.privateNets, n)
	}

	_, bl.loopback, _ = net.ParseCIDR("127.0.0.0/8")
	_, bl.linkLocal, _ = net.ParseCIDR("169.254.0.0/16")
	return bl
}

// IsBlocked returns true if the IP is in any internal or non-routable range.
func (bl *IPBlocklist) IsBlocked(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Also check the Go 1.17+ IsPrivate if available, but we manually check
	// RFC 1918 ranges for older Go versions.
	for _, n := range bl.privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return bl.loopback.Contains(ip) || bl.linkLocal.Contains(ip)
}
```

### 6.2 Hardened HTTP Transport

The transport below combines the `SafeDialer` (section 4.3) with the
`IPBlocklist` (section 6.1) to produce an `http.Client` that cannot connect to
internal IP addresses regardless of DNS:

```go
// ValidateURL performs pre-flight validation on a URL before making an HTTP
// request. It checks the scheme, the hostname (not an IP literal), and resolves
// the hostname to verify that at least one IP is not blocked.
func ValidateURL(ctx context.Context, rawURL string, bl *IPBlocklist) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}

	// Only allow http and https schemes — prevent file://, gopher://, etc.
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme %q not allowed", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	// If the host is an IP literal, validate immediately.
	if ip := net.ParseIP(host); ip != nil {
		if bl.IsBlocked(ip) {
			return fmt.Errorf("target IP %s is in blocked range", ip)
		}
		return nil
	}

	// For hostnames, resolve and check all candidate IPs.
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("DNS lookup %s: %w", host, err)
	}

	anyAllowed := false
	for _, ipAddr := range ips {
		if !bl.IsBlocked(ipAddr.IP) {
			anyAllowed = true
			break
		}
	}
	if !anyAllowed {
		return fmt.Errorf("all IPs for %s are in blocked ranges", host)
	}

	return nil
}
```

### 6.3 Usage with SAML Metadata Fetch

```go
func FetchSAMLMetadata(ctx context.Context, metadataURL string) (*saml.Metadata, error) {
	bl := ssrf.DefaultBlocklist()

	// Pre-flight: validate the URL and its resolved IPs.
	if err := ssrf.ValidateURL(ctx, metadataURL, bl); err != nil {
		return nil, fmt.Errorf("metadata URL validation: %w", err)
	}

	// Use a hardened client that will also validate at connection time.
	client := &http.Client{
		Transport: ssrf.SafeTransport(),
		Timeout:   15 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata fetch returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, err
	}

	var meta saml.Metadata
	if err := xml.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &meta, nil
}
```

---

## 7. Webhook URL Validation

### 7.1 Why Webhook Delivery Is a DNS Rebinding/SSRF Vector

Webhook URLs are **user-controlled**. Any tenant admin can register a webhook
URL pointing to an arbitrary destination. The IAM service then makes an
outbound HTTP POST to that URL whenever a matching event fires. This creates
a classic SSRF vector:

- Register a webhook for `http://169.254.169.254/latest/meta-data/iam/security-credentials/role-name`
  to exfiltrate cloud credentials.
- Register a webhook for `http://10.0.0.5:8080/admin` to access internal
  services.
- Register a webhook for `http://localhost:6379/` to interact with Redis
  (though Redis speaks RESP, not HTTP, some proxies interpret it).
- Use DNS rebinding: register a webhook for a hostname that initially resolves
  to an external IP (passing validation) but later resolves to `169.254.169.254`.

### 7.2 Safe Webhook Delivery Implementation

```go
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ggid/pkg/ssrf"
)

// SafeDeliverer is an HTTPDeliverer that validates webhook URLs before delivery
// and uses a hardened transport that blocks internal IP ranges.
type SafeDeliverer struct {
	client     *http.Client
	blocklist  *ssrf.IPBlocklist
	maxRetries int
	maxBody    int64 // max response body to read (bytes)
}

func NewSafeDeliverer() *SafeDeliverer {
	bl := ssrf.DefaultBlocklist()
	return &SafeDeliverer{
		client: &http.Client{
			Transport: ssrf.SafeTransport(),
			Timeout:   10 * time.Second,
			// Do not follow redirects — a redirect could point to 169.254.169.254.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		blocklist:  bl,
		maxRetries: 3,
		maxBody:    0, // do not read response body for webhooks
	}
}

// ValidateWebhookURL checks that a webhook URL is safe to deliver to.
// This should be called at registration time AND at delivery time (TOCTOU).
func ValidateWebhookURL(ctx context.Context, rawURL string, bl *ssrf.IPBlocklist) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}

	// 1. Only HTTPS or HTTP (HTTPS strongly recommended in production).
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("scheme %q not allowed", u.Scheme)
	}

	// 2. Block userinfo (credentials in URL).
	if u.User != nil {
		return fmt.Errorf("userinfo not allowed in webhook URL")
	}

	// 3. Host must be present.
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	// 4. If IP literal, check blocklist.
	if ip := net.ParseIP(host); ip != nil {
		if bl.IsBlocked(ip) {
			return fmt.Errorf("webhook target IP %s is blocked", ip)
		}
	} else {
		// 5. Resolve hostname and check ALL resolved IPs.
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return fmt.Errorf("DNS resolution: %w", err)
		}
		for _, ipAddr := range ips {
			if bl.IsBlocked(ipAddr.IP) {
				return fmt.Errorf("webhook target %s resolves to blocked IP %s",
					host, ipAddr.IP)
			}
		}
	}

	// 6. Block localhost variants.
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") {
		return fmt.Errorf("localhost targets not allowed")
	}

	return nil
}

// Deliver sends a webhook payload with HMAC-SHA256 signature.
func (d *SafeDeliverer) Deliver(ctx context.Context, webhookURL, secret string, payload []byte) error {
	// Validate URL at delivery time (defense-in-depth against TOCTOU).
	if err := ValidateWebhookURL(ctx, webhookURL, d.blocklist); err != nil {
		return fmt.Errorf("webhook URL rejected: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < d.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*attempt) * time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "GGID-Webhook/1.0")

		// HMAC-SHA256 signature.
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(payload)
			sig := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-GGID-Signature", "sha256="+sig)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Discard response body — webhook receivers should not be able to
		// exfiltrate data via large response bodies.
		if d.maxBody > 0 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, d.maxBody))
		} else {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return lastErr
}
```

### 7.3 Registration-Time Validation

The `Create` handler should validate the webhook URL at registration time, not
just at delivery time:

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// ... existing parsing ...

	if err := ValidateWebhookURL(r.Context(), req.URL, h.blocklist); err != nil {
		writeJSON(w, 400, map[string]string{
			"error": fmt.Sprintf("invalid webhook URL: %v", err),
		})
		return
	}

	wh := &Webhook{
		ID:       uuid.NewString(),
		TenantID: tenantID,
		URL:      req.URL,
		Events:   req.Events,
		Secret:   req.Secret,
		Active:   true,
	}
	// ... store webhook ...
}
```

---

## 8. GGID Gateway Host Header Analysis

### 8.1 Gateway Router Host Header Handling

**File**: `services/gateway/internal/router/router.go`

The gateway uses `httputil.NewSingleHostReverseProxy(parsed)` for each backend
route. The reverse proxy's default `Director` sets `req.Host = target.Host` for
the upstream request, but **does not validate the incoming `r.Host`**. The
`ServeHTTP` method dispatches based on `r.URL.Path` only — there is no Host
header check before routing.

```go
// Current code (simplified):
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Health checks, metrics, JWKS, docs — all path-based
    if r.URL.Path == "/healthz" { ... }

    // Route to backend — no Host validation
    for prefix, proxy := range gw.proxies {
        if strings.HasPrefix(r.URL.Path, prefix) {
            proxy.ServeHTTP(w, r)
            return
        }
    }
}
```

**Finding**: No Host header validation exists. A request with
`Host: evil.example.com` is routed normally.

### 8.2 Tenant Middleware Host Usage

**File**: `services/gateway/internal/middleware/tenant_enhanced.go`

The enhanced tenant resolver reads `r.Host` for subdomain extraction:

```go
host := r.Host
if idx := strings.LastIndex(host, ":"); idx != -1 {
    host = host[:idx]
}
if strings.HasSuffix(host, cfg.DomainSuffix) {
    sub := strings.TrimSuffix(host, cfg.DomainSuffix)
    // Extract tenant from subdomain prefix
}
```

**Finding**: The tenant resolver *uses* the Host header but does not *validate*
it. An attacker can set `Host: acme.iam.ggid.io` to spoof tenant resolution,
even if the actual request arrives via DNS rebinding from `evil.example.com`.

**Risk**: Medium. If tenant aliases map to different data, an attacker could
impersonate one tenant by setting a spoofed Host header.

### 8.3 Webhook Delivery Security

**File**: `services/gateway/internal/webhooks/webhooks.go`

The current `HTTPDeliverer` implementation:

```go
func NewHTTPDeliverer() *HTTPDeliverer {
    return &HTTPDeliverer{
        client:     &http.Client{Timeout: 10 * time.Second},
        maxRetries: 3,
    }
}

func (d *HTTPDeliverer) Deliver(ctx context.Context, url, secret string, payload []byte) error {
    // No URL validation!
    // No IP range check!
    // No redirect restriction!
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
    // ...
    resp, err := d.client.Do(req)
    // ...
}
```

**Findings**:
1. **No URL validation** — the webhook `Create` handler only checks
   `req.URL == ""`. Any URL scheme (including `file://`, `gopher://` if Go
   allowed them) is accepted.
2. **No IP blocklist** — a webhook URL pointing to `http://169.254.169.254/`
   or `http://10.0.0.1/` is accepted and will be called.
3. **No redirect restriction** — Go's `http.Client` follows redirects by
   default. An attacker can register a webhook pointing to an external URL
   that 302-redirects to `169.254.169.254`.
4. **Response body is read** — `resp.Body.Close()` is called but the body
   could still be exfiltrated if the deliverer processed responses.

**Risk**: Critical. This is the most significant SSRF vector in GGID.

### 8.4 SAML Metadata Fetch

**File**: `pkg/saml/sp.go`

The SAML package generates SP metadata (`GenerateSPMetadata`) but does **not**
fetch IdP metadata from URLs. The `BuildAuthnRequest` function takes an
`idpSSOURL` parameter as a string — there is no outbound fetch.

**Finding**: No IdP metadata fetch exists in `pkg/saml`. However, the auth
service or admin console will eventually need to fetch IdP metadata when an
admin configures a new SAML integration. When this feature is implemented, it
must use the SSRF-safe transport described in section 6.3.

**Risk**: Latent. The vulnerability will be introduced when SAML IdP metadata
fetching is added without SSRF protection.

### 8.5 Social Login API Calls

**File**: `pkg/social/github.go`, `pkg/social/microsoft.go`

These files make hardcoded calls to `https://api.github.com/user` and
`https://graph.microsoft.com/v1.0/me`. The URLs are constants, not
user-controlled.

**Finding**: Safe. Hardcoded URLs cannot be redirected via DNS rebinding unless
the system's DNS resolver is compromised at the OS level, which is outside the
application threat model.

---

## 9. Gap Analysis and Recommendations

### 9.1 Current Security Posture Summary

| Component | Host Header Validation | SSRF Protection | DNS Pinning | Risk |
|-----------|----------------------|-----------------|-------------|------|
| Gateway router | None | N/A | N/A | High |
| Tenant resolver | Uses Host, does not validate | N/A | N/A | Medium |
| Webhook delivery | N/A | None | None | **Critical** |
| SAML metadata fetch | N/A | Not yet implemented | N/A | Latent |
| Social login | N/A | Hardcoded URLs (safe) | N/A | Low |

### 9.2 Identified Gaps

**Gap 1: No Host Header Allowlist on Gateway**
The gateway accepts requests with any Host header value. DNS rebinding attacks
from any web page the user visits can reach internal IAM APIs. Even without
rebinding, Host header spoofing enables tenant impersonation via the subdomain
tenant resolver.

**Gap 2: Webhook Delivery Has Zero SSRF Protection**
`HTTPDeliverer` uses a plain `http.Client` with no URL validation, no IP
blocklist, and no redirect restriction. A tenant admin can register a webhook
to `http://169.254.169.254/` and exfiltrate cloud credentials.

**Gap 3: No Internal Service Authentication Model**
GGID backend services (identity, auth, policy, org, audit) trust requests from
the gateway without verifying that they actually came through the gateway. If an
attacker can reach a backend service directly (via DNS rebinding to
`localhost:8081`), they bypass all gateway-level authentication.

**Gap 4: No DNS Resolution Hardening in Outbound Clients**
No `http.Transport` in the codebase uses a custom `DialContext` that validates
resolved IPs. All outbound HTTP requests are vulnerable to TOCTOU DNS races.

**Gap 5: No SAML IdP Metadata Fetch Protection (Latent)**
SAML IdP metadata fetch is not yet implemented. When added, it will be an SSRF
vector unless the hardened transport from section 6 is used.

### 9.3 Implementation Roadmap

| # | Action Item | Effort | Priority | Files Affected |
|---|------------|--------|----------|----------------|
| 1 | **Add Host header allowlist middleware to gateway** | Small (2-4h) | P0 | `services/gateway/internal/middleware/host_allowlist.go` (new), `services/gateway/internal/router/router.go` (wire middleware) |
| 2 | **Replace HTTPDeliverer with SafeDeliverer** | Medium (4-6h) | P0 | `services/gateway/internal/webhooks/webhooks.go`, add `pkg/ssrf/` package |
| 3 | **Create `pkg/ssrf` package with IPBlocklist + SafeTransport** | Medium (4-6h) | P0 | `pkg/ssrf/ssrf.go` (new), `pkg/ssrf/blocklist.go` (new) |
| 4 | **Add mutual TLS or shared-secret between gateway and backends** | Large (1-2d) | P1 | All `services/*/cmd/main.go`, gateway router director |
| 5 | **Implement SSRF-safe SAML IdP metadata fetch** | Small (2-4h) | P2 | `pkg/saml/idp.go` (new), use `pkg/ssrf` transport |

### 9.4 Recommended Implementation Order

**Phase 1 — Immediate (P0):**
1. Create `pkg/ssrf` package with `IPBlocklist`, `SafeDialer`, `SafeTransport`,
   `ValidateURL`, and `SafeHTTPClient` (sections 4.3, 6.1, 6.2).
2. Create `HostAllowlistMiddleware` in gateway middleware (section 5.1).
3. Wire the middleware as the outermost layer in the gateway's middleware chain.
4. Replace `HTTPDeliverer` with `SafeDeliverer` in webhook package (section 7.2).
5. Add URL validation to webhook `Create` handler (section 7.3).

**Phase 2 — Near-term (P1):**
6. Add shared-secret header validation between gateway and backend services.
   The gateway already sets `X-Tenant-ID` and `X-User-ID` — add an
   `X-GGID-Internal-Token` that backends validate.
7. Enable Go 1.17+ `net.IP.IsPrivate()` in blocklist for forward compatibility.

**Phase 3 — Future (P2):**
8. When SAML IdP metadata fetch is implemented, use `pkg/ssrf` transport.
9. Add integration test that verifies webhook delivery to `169.254.169.254`
   is rejected.
10. Add integration test that verifies Host header allowlist blocks
    `evil.example.com`.

### 9.5 Configuration Recommendations

```yaml
# ggateway.yaml additions
host_allowlist:
  enabled: true
  allowed_hosts:
    - iam.ggid.io
    - api.ggid.io
    - console.ggid.io
    - localhost
    - 127.0.0.1
  allowed_suffixes:
    - ".iam.ggid.io"
  trust_proxy: true       # behind ALB/NGINX
  allow_empty_host: false

webhook:
  require_https: true     # production: reject http:// webhook URLs
  max_redirects: 0        # never follow redirects
  response_body_limit: 0  # do not read webhook response bodies
  block_internal_ips: true
```

---

## Appendix A: DNS Rebinding Detection Signals

For organizations with WAF or IDS capabilities, the following signals indicate
DNS rebinding attempts:

1. **Rapid DNS TTL changes** — a hostname that resolves to different IPs within
   seconds, with TTL < 60s.
2. **Host header mismatch** — incoming HTTP request has `Host: X` but the
   resolved IP belongs to a different organization's ASN.
3. **Unusual internal IP targets** — outbound connections from application
   servers to `169.254.169.254`, `127.0.0.1`, or RFC 1918 ranges that are not
   in the expected service mesh topology.
4. **Short-lived DNS records** — A records with TTL=0 or TTL=1 that change
   frequently are a hallmark of DNS rebinding infrastructure.

## Appendix B: References

- **RFC 1918** — Address Allocation for Private Internets
- **RFC 3927** — Dynamic Configuration of IPv4 Link-Local Addresses (169.254.0.0/16)
- **RFC 7230 §5.4** — Host header semantics (must reject multiple Host headers)
- **AWS IMDSv2** — Session-token-based protection against SSRF to metadata
- **OWASP SSRF Prevention Cheat Sheet** — Defense-in-depth patterns
- **Stanford DNS Rebinding Research** — https://dnsrebinding.org
- **CVE-2018-17144** — geth JSON-RPC rebinding attacks on Ethereum wallets
- **Go `net.IP` documentation** — `IsPrivate()`, `IsLoopback()`, `IsLinkLocalUnicast()`

---

*Document version: 1.0 | Last updated: 2025 | GGID IAM Suite Security Research*
