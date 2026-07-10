# IP Reputation and Threat Intelligence for IAM Systems

> **Document type:** Security Research
> **Project:** GGID IAM Suite
> **Author:** Security Research Team
> **Status:** Draft for Review

---

## Table of Contents

1. [IP Threat Categories](#1-ip-threat-categories)
2. [Tor Exit Node Detection](#2-tor-exit-node-detection)
3. [VPN/Proxy Identification](#3-vpnproxy-identification)
4. [Geo-Threat Intelligence](#4-geo-threat-intelligence)
5. [IP Reputation Scoring Algorithm](#5-ip-reputation-scoring-algorithm)
6. [fail2ban Integration Patterns](#6-fail2ban-integration-patterns)
7. [Distributed IP Block List](#7-distributed-ip-block-list)
8. [GGID Gateway IP Filter Design](#8-ggid-gateway-ip-filter-design)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. IP Threat Categories

Every request that reaches an IAM gateway originates from an IP address, and the reputation of that address is the single most powerful first-pass signal for distinguishing legitimate users from automated threats. This section categorises the IP-based threats most relevant to identity and access management systems.

### 1.1 Known Malicious IPs (Botnets, Scanners)

Botnet-controlled IPs are the primary source of **credential stuffing** attacks against IAM systems. Attackers rent botnets — collections of compromised devices — to distribute login attempts across thousands of IPs, each sending only a few requests to evade per-IP rate limits. Common patterns include:

- **Mirai-class IoT botnets** — Compromised routers, cameras, and DVRs. These IPs are typically residential broadband connections, making them hard to distinguish from legitimate users without threat intel feeds.
- **Scanner IPs** — Tools like Masscan, ZMap, and Nmap scan the entire IPv4 address space in minutes. Security search engines (Shodan, Censys, ZoomEye) index open ports. Requests from scanner IPs are reconnaissance, not legitimate auth.
- **Credential stuffing botnets** — These rotate through curated IP lists, often combining residential proxies with automation frameworks like Sentry MBA, SNIPR, or OpenBullet.

**IAM relevance:** A credential stuffing attack with 50,000 username/password pairs spread across 10,000 IPs averages only 5 attempts per IP — below any reasonable per-IP rate limit threshold. Only aggregate reputation scoring across all requests can detect this pattern.

### 1.2 Tor Exit Nodes

The Tor network provides anonymity by routing traffic through three volunteer-operated relays. The **exit node** is the last relay — the one that connects to the destination server and is visible in server logs.

- There are approximately 1,000–2,000 active Tor exit nodes at any given time.
- The Tor Project publishes a bulk exit node list that can be queried by destination port.
- Tor traffic is used for: anonymous browsing (legitimate), but also brute force, credential stuffing, and account takeover attempts.

**IAM relevance:** Very few legitimate users log into enterprise IAM systems via Tor. Blocking Tor exit nodes for authentication endpoints (login, register, password reset) is a low-friction, high-impact security measure. For OIDC/OAuth flows, blocking Tor at the token endpoint eliminates most automated token abuse.

### 1.3 VPN and Proxy Detection

VPNs are a double-edged sword for IAM. Many users legitimately access corporate services via VPN (remote workers, travelling employees). However, VPN and proxy IPs are also the primary infrastructure for:

- **Geo-evasion** — Attackers use VPNs to appear from a trusted geography.
- **IP rotation** — Commercial proxy services (Bright Data, Oxylabs) offer rotating residential IPs that change every request.
- **Anonymity** — VPNs add a layer between the attacker and the target, complicating attribution.

**IAM relevance:** VPN detection should not be a binary block/allow. Instead, VPN/proxy should be one signal in a multi-factor risk score. A known data-center VPN (e.g., a corporate VPN endpoint) can be allowlisted, while a residential proxy service used by a credential stuffing botnet should be flagged.

### 1.4 Residential Proxy Networks

Residential proxies are IP addresses belonging to real ISPs (Comcast, AT&T, Verizon) that are rented out as proxies. They are the most dangerous category because they:

- Appear as normal residential traffic — not flagged by ASN-based data-center detection.
- Rotate frequently — each request can come from a different residential IP.
- Are created by: compromised devices (botnet), browser extensions that share bandwidth, or SDK-integrated apps that proxy traffic.

Commercial services like Luminati/Bright Data, Smartproxy, and Soax provide legitimate API access to these networks. Attackers also build their own via malware.

**IAM relevance:** Residential proxies are the #1 challenge for credential stuffing defense. Per-IP rate limits are bypassed. The only effective countermeasures are: device fingerprinting, behavioural analysis (request timing, mouse movement, CAPTCHA), and velocity checks across the tenant's entire IP pool.

### 1.5 Cloud Provider IPs (Data Center Traffic)

Cloud provider IP ranges (AWS, GCP, Azure, DigitalOcean, Hetzner, OVH) indicate **data center traffic**. Legitimate users almost never browse from a data center IP. When an auth request comes from an AWS IP, it is almost certainly:

- A bot/script running on a cloud server.
- A credential stuffing tool deployed on cloud infrastructure.
- A web scraper testing endpoints.
- Rarely: a corporate VPN hosted in the cloud.

ASN-based detection of data center ranges is highly effective because cloud providers publish their IP ranges publicly (AWS, GCP, Azure).

**IAM relevance:** Flagging all cloud-provider IPs with elevated risk scores on auth endpoints catches the majority of automated attacks. False positive rate is low for auth endpoints (login/register) and higher for API endpoints (which may receive legitimate server-to-server traffic).

### 1.6 Compromised Host Indicators

Individual host compromise indicators are harder to detect at the network level but include:

- **Reverse DNS anomalies** — A "comcast.net" PTR record from a known botnet IP.
- **Open port signatures** — Ports 22 (SSH), 3389 (RDP), 5900 (VNC) open on the host.
- **Threat intel feed hits** — IP appears in Spamhaus, AbuseIPDB, or AlienVault OTX.
- **Historical auth failure rate** — The IP has N failed logins in the past hour.

---

## 2. Tor Exit Node Detection

### 2.1 Tor Bulk Exit List API

The Tor Project publishes a real-time exit node list at:

```
https://check.torproject.org/torbulkexitlist
```

This returns a plain-text list of IP addresses, one per line. To query nodes that can exit to a specific port:

```
https://check.torproject.org/api/bulk exit-list?ip=<your_server_ip>&port=443
```

For IAM systems, the most relevant ports are:

| Port | Service | Query |
|------|---------|-------|
| 443  | HTTPS login/token | `&port=443` |
| 80   | HTTP redirect | `&port=80` |

### 2.2 Caching Strategy

- **Refresh interval:** Hourly (Tor exits change slowly, ~50-100 nodes/day).
- **Storage:** In-memory `map[string]bool` for O(1) lookups.
- **Fallback:** If the fetch fails, continue using the last successful list.
- **Graceful degradation:** If no list is available, do not block (fail-open to avoid DoS).

### 2.3 Go Implementation

```go
package threatintel

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TorExitChecker maintains a cached set of Tor exit node IPs.
type TorExitChecker struct {
	mu          sync.RWMutex
	exitNodes   map[string]bool
	lastUpdated time.Time
	client      *http.Client
	fetchURL    string
}

// NewTorExitChecker creates a Tor exit node checker that refreshes hourly.
func NewTorExitChecker() *TorExitChecker {
	return &TorExitChecker{
		exitNodes: make(map[string]bool),
		client:    &http.Client{Timeout: 10 * time.Second},
		fetchURL:  "https://check.torproject.org/torbulkexitlist",
	}
}

// StartRefresh launches a background goroutine that refreshes the list.
func (t *TorExitChecker) StartRefresh(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		t.Refresh() // initial fetch
		for {
			select {
			case <-ticker.C:
				t.Refresh()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Refresh fetches the latest Tor exit node list.
func (t *TorExitChecker) Refresh() error {
	resp, err := t.client.Get(t.fetchURL)
	if err != nil {
		return fmt.Errorf("tor exit list fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tor exit list returned %d", resp.StatusCode)
	}

	newList := make(map[string]bool)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" {
			newList[ip] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("tor exit list parse error: %w", err)
	}

	t.mu.Lock()
	t.exitNodes = newList
	t.lastUpdated = time.Now()
	t.mu.Unlock()

	return nil
}

// IsTorExit checks if an IP is a known Tor exit node.
func (t *TorExitChecker) IsTorExit(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.exitNodes[ip]
}

// Count returns the number of cached exit nodes (for monitoring).
func (t *TorExitChecker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.exitNodes)
}

// LastUpdated returns the timestamp of the last successful refresh.
func (t *TorExitChecker) LastUpdated() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastUpdated
}
```

### 2.4 Integration with GGID Gateway

The checker should be initialized in the gateway `main()` function and passed to the middleware:

```go
torChecker := threatintel.NewTorExitChecker()
torChecker.StartRefresh(ctx)
handler = middleware.TorBlockMiddleware(torChecker)(handler)
```

The middleware blocks Tor exit nodes on auth-specific paths only:

```go
func TorBlockMiddleware(checker *TorExitChecker) func(http.Handler) http.Handler {
	authPaths := []string{"/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/reset"}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range authPaths {
				if strings.HasPrefix(r.URL.Path, p) {
					ip := ClientIP(r)
					if checker.IsTorExit(ip) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						json.NewEncoder(w).Encode(map[string]any{
							"error":   "tor_blocked",
							"message": "Tor exit nodes are not allowed on authentication endpoints",
						})
						return
					}
					break
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

---

## 3. VPN/Proxy Identification

### 3.1 Commercial API Providers

| Provider | Detection | Pricing | Key Features |
|----------|-----------|---------|--------------|
| **MaxMind GeoIP2** | Geo + ISP + Proxy | $ per query | Industry standard, GeoLite2 (free tier) |
| **IPinfo** | Proxy/VPN + ASN | $ per query | Generous free tier (50K/mo), fast API |
| **IPQualityScore** | Proxy + VPN + Fraud | $ per query | Fraud score, residential proxy detection |
| **Abstract API** | Proxy + VPN | $ per query | Simple REST API |

### 3.2 Free Alternatives

| Provider | Detection | Limitations |
|----------|-----------|-------------|
| **IP2Location Lite** | Proxy + Country | 2 updates/year, no real-time |
| **MaxMind GeoLite2** | Country/City + ASN | No proxy flag in free tier |
| **AbuseIPDB** | Abuse reports | Community-driven, rate limited |
| **AlienVault OTX** | Threat pulses | Requires manual integration |

### 3.3 Detection Signals (Without External APIs)

When commercial APIs are unavailable, the following signals can be computed locally:

1. **ASN type** — If the ASN is a known hosting provider (AWS AS16509, GCP AS15169, Azure AS8075, DigitalOcean AS14061, Hetzner AS24940, OVH AS16276), flag as data center.
2. **PTR record** — If the reverse DNS contains "cloud", "hosted", "server", "datacenter", "vps", "dedicated".
3. **Open proxy ports** — Ports 1080 (SOCKS), 3128 (Squid), 8080 (HTTP proxy) detected by the connecting host.
4. **IP range overlap** — Cross-reference with published cloud provider CIDR ranges.

### 3.4 Go Implementation

```go
package threatintel

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VPNChecker provides VPN/proxy detection via multiple signals.
type VPNChecker struct {
	mu           sync.RWMutex
	dataCenterASNs map[uint32]bool // ASN numbers for cloud providers
	httpClient   *http.Client
	// Optional: external API config
	apiKey   string
	provider string // "ipinfo", "ipqualityscore", ""
}

// NewVPNChecker creates a VPN checker with built-in cloud ASN ranges.
func NewVPNChecker() *VPNChecker {
	// Known data-center ASN numbers
	dcASNs := map[uint32]bool{
		16509: true, // AWS
		14618: true, // AWS (us-east)
		15169: true, // Google
		396982: true, // Google Cloud
		8075: true,  // Microsoft Azure
		8068: true,  // Microsoft Azure
		14061: true, // DigitalOcean
		24940: true, // Hetzner
		16276: true, // OVH
		60068: true, // CDN77 / Datacamp
		9009: true,  // M247 (budget VPS)
		62240: true, // Clouvider
	}
	return &VPNChecker{
		dataCenterASNs: dcASNs,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

// VPNResult holds the output of a VPN/proxy check.
type VPNResult struct {
	IsVPN        bool
	IsDataCenter bool
	IsProxy      bool
	ASN          uint32
	Org          string
	Country      string
	RiskScore    int // 0-100
}

// Check evaluates whether an IP is a VPN/proxy/data-center.
func (v *VPNChecker) Check(ip string) *VPNResult {
	result := &VPNResult{}

	// 1. Check PTR record for hosting indicators
	ptr := v.lookupPTR(ip)
	if v.ptrLooksLikeHosting(ptr) {
		result.IsDataCenter = true
		result.RiskScore += 30
	}

	// 2. Check against data-center ASN list (requires ASN lookup)
	asn, org := v.lookupASN(ip)
	result.ASN = asn
	result.Org = org
	if v.dataCenterASNs[asn] {
		result.IsDataCenter = true
		result.RiskScore += 40
	}

	// 3. If external API configured, query it
	if v.apiKey != "" && v.provider != "" {
		if extResult := v.queryExternalAPI(ip); extResult != nil {
			if extResult["proxy"].(bool) {
				result.IsProxy = true
				result.RiskScore += 35
			}
			if extResult["vpn"].(bool) {
				result.IsVPN = true
				result.RiskScore += 25
			}
		}
	}

	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	return result
}

// lookupPTR performs a reverse DNS lookup.
func (v *VPNChecker) lookupPTR(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

// ptrLooksLikeHosting checks if a PTR record indicates a hosting provider.
func (v *VPNChecker) ptrLooksLikeHosting(ptr string) bool {
	indicators := []string{
		"amazonaws", "google", "azure", "cloud",
		"hosted", "server", "datacenter", "vps",
		"dedicated", "ovh", "hetzner", "digitalocean",
		"linode", "vultr", "kimsufi", "scaleway",
	}
	lower := strings.ToLower(ptr)
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}

// lookupASN is a stub — in production, use a MaxMind GeoLite2 ASN database.
func (v *VPNChecker) lookupASN(ip string) (uint32, string) {
	// In production: open GeoLite2-ASN.mmdb and look up ip
	return 0, ""
}

// queryExternalAPI queries a commercial VPN detection API.
func (v *VPNChecker) queryExternalAPI(ip string) map[string]any {
	var url string
	switch v.provider {
	case "ipinfo":
		url = fmt.Sprintf("https://ipinfo.io/%s/json?token=%s", ip, v.apiKey)
	case "ipqualityscore":
		url = fmt.Sprintf("https://www.ipqualityscore.com/api/json/ip/%s/%s", v.apiKey, ip)
	default:
		return nil
	}

	resp, err := v.httpClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result
}

// SetExternalAPI configures an external VPN detection provider.
func (v *VPNChecker) SetExternalAPI(provider, apiKey string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.provider = provider
	v.apiKey = apiKey
}
```

---

## 4. Geo-Threat Intelligence

### 4.1 Impossible Travel Detection

Impossible travel (also called "velocity check") detects when the same user appears to log in from two geographically distant locations in an impossibly short time frame. For example:

- Login from New York at 14:00 UTC
- Login from Beijing at 14:30 UTC

The great-circle distance between NYC and Beijing is ~11,000 km. To travel that in 30 minutes requires a speed of ~22,000 km/h — far beyond commercial aircraft (~900 km/h). This is impossible travel, indicating either a compromised account or a session hijack.

### 4.2 Country-Level Risk Tiers

```go
var countryRiskTiers = map[string]string{
	// Tier 1: Low risk — established rule-of-law, low cybercrime
	"US": "low", "CA": "low", "GB": "low", "AU": "low", "NZ": "low",
	"DE": "low", "FR": "low", "NL": "low", "SE": "low", "NO": "low",
	"JP": "low", "SG": "low", "CH": "low",

	// Tier 2: Medium risk — moderate threat actor presence
	"BR": "medium", "IN": "medium", "ID": "medium", "VN": "medium",
	"TH": "medium", "PH": "medium", "RU": "medium", "UA": "medium",
	"TR": "medium", "MX": "medium", "EG": "medium", "ZA": "medium",

	// Tier 3: High risk — known state-sponsored actors, high cybercrime
	"CN": "high", "KP": "high", "IR": "high", "BY": "high",
	"NG": "high", "PK": "high", "BD": "high",
}
```

**Note:** Country risk tiers are policy decisions. GGID should expose them as tenant-configurable settings, not hard-code them in the binary.

### 4.3 Velocity Checks Per Geography

Beyond impossible travel for a single user, velocity checks monitor aggregate patterns:

- **New country burst** — If a tenant suddenly sees logins from 20+ new countries in 1 hour, it indicates a credential stuffing campaign using a distributed proxy network.
- **Geographic clustering** — Auth failures concentrated in a specific ASN/country indicate a targeted attack from that region.

### 4.4 Go Implementation

```go
package threatintel

import (
	"math"
	"sync"
	"time"
)

// GeoAnomalyDetector detects impossible travel and geo-velocity anomalies.
type GeoAnomalyDetector struct {
	mu          sync.Mutex
	sessions    map[string][]geoLoginEvent // keyed by user ID
	maxSpeedKmh float64                     // max plausible travel speed
}

type geoLoginEvent struct {
	timestamp time.Time
	lat       float64
	lon       float64
	country   string
	ip        string
}

// NewGeoAnomalyDetector creates a detector with the given max-speed threshold.
// Default: 900 km/h (commercial jet speed + margin).
func NewGeoAnomalyDetector(maxSpeedKmh float64) *GeoAnomalyDetector {
	if maxSpeedKmh <= 0 {
		maxSpeedKmh = 900
	}
	return &GeoAnomalyDetector{
		sessions:    make(map[string][]geoLoginEvent),
		maxSpeedKmh: maxSpeedKmh,
	}
}

// RecordLogin records a login event and returns whether impossible travel was detected.
func (g *GeoAnomalyDetector) RecordLogin(
	userID string, lat, lon float64, country, ip string,
) (impossibleTravel bool, speedKmh float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	event := geoLoginEvent{
		timestamp: now,
		lat:       lat,
		lon:       lon,
		country:   country,
		ip:        ip,
	}

	events := g.sessions[userID]
	if len(events) > 0 {
		last := events[len(events)-1]
		dist := haversine(last.lat, last.lon, lat, lon)
		timeDiff := now.Sub(last.timestamp).Hours()
		if timeDiff > 0 {
			speed := dist / timeDiff
			if speed > g.maxSpeedKmh {
				return true, speed
			}
		}
	}

	// Keep last 10 events per user
	if len(events) >= 10 {
		events = events[1:]
	}
	g.sessions[userID] = append(events, event)

	return false, 0
}

// haversine computes the great-circle distance in km between two lat/lon points.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)

	lat1 = toRad(lat1)
	lat2 = toRad(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// CountryVelocity tracks login counts per country to detect bursts.
type CountryVelocity struct {
	mu       sync.Mutex
	counters map[string]map[string]int // tenantID → country → count
	window   time.Duration
}

// NewCountryVelocity creates a velocity tracker with a rolling window.
func NewCountryVelocity(window time.Duration) *CountryVelocity {
	return &CountryVelocity{
		counters: make(map[string]map[string]int),
		window:   window,
	}
}

// Record increments the login count for a tenant/country and returns the count.
func (cv *CountryVelocity) Record(tenantID, country string) int {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	if cv.counters[tenantID] == nil {
		cv.counters[tenantID] = make(map[string]int)
	}
	cv.counters[tenantID][country]++
	return cv.counters[tenantID][country]
}

// GetThreshold returns the velocity for a tenant/country.
func (cv *CountryVelocity) Get(tenantID, country string) int {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if cv.counters[tenantID] == nil {
		return 0
	}
	return cv.counters[tenantID][country]
}

// Reset clears all counters (called periodically).
func (cv *CountryVelocity) Reset() {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	cv.counters = make(map[string]map[string]int)
}
```

---

## 5. IP Reputation Scoring Algorithm

### 5.1 Design Principles

The IP reputation score combines multiple weak signals into a single strong signal:

| Signal | Weight | Notes |
|--------|--------|-------|
| Threat intel feed hits (AbuseIPDB, OTX) | 35 | Authoritative abuse reports |
| Tor exit node | 25 | Anonymity network on auth endpoints |
| VPN/data center (ASN-based) | 20 | Non-residential traffic |
| Geo anomaly (impossible travel) | 15 | Session hijack indicator |
| Velocity (request rate per IP) | 15 | Brute force / stuffing |
| Auth failure history (last hour) | 20 | Direct evidence of abuse |
| Country risk tier | 10 | Policy-based geo restriction |

Weights are additive but capped at 100. The final score determines the action:

| Score | Action | Description |
|-------|--------|-------------|
| 0–29 | Allow | No action, normal flow |
| 30–59 | Challenge | Require CAPTCHA, step-up MFA, or device verification |
| 60–79 | Throttle | Reduce rate limit, add delay, flag for review |
| 80–100 | Block | 403 Forbidden, log to audit |

### 5.2 Go Implementation

```go
package threatintel

import "sync"

// IPReputationScorer combines multiple signals into a 0-100 risk score.
type IPReputationScorer struct {
	torChecker    *TorExitChecker
	vpnChecker    *VPNChecker
	geoDetector   *GeoAnomalyDetector
	failureStore  *AuthFailureStore
	threatFeeds   ThreatFeedProvider
}

// ThreatFeedProvider checks external threat intelligence feeds.
type ThreatFeedProvider interface {
	IsListed(ip string) bool
	GetScore(ip string) int // 0-100 from the feed
}

// AuthFailureStore tracks recent authentication failures per IP.
type AuthFailureStore struct {
	mu      sync.Mutex
	failures map[string]*failureTracker
	window   int // seconds
}

type failureTracker struct {
	count     int
	firstSeen int64 // unix timestamp
	lastSeen  int64
}

// NewAuthFailureStore creates a store tracking failures within a time window.
func NewAuthFailureStore(windowSeconds int) *AuthFailureStore {
	if windowSeconds <= 0 {
		windowSeconds = 3600 // 1 hour default
	}
	return &AuthFailureStore{
		failures: make(map[string]*failureTracker),
		window:   windowSeconds,
	}
}

// RecordFailure records an auth failure for an IP.
func (s *AuthFailureStore) RecordFailure(ip string, now int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ft, ok := s.failures[ip]
	if !ok {
		s.failures[ip] = &failureTracker{count: 1, firstSeen: now, lastSeen: now}
		return
	}
	ft.count++
	ft.lastSeen = now
}

// GetFailureCount returns the failure count for an IP within the window.
func (s *AuthFailureStore) GetFailureCount(ip string, now int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ft, ok := s.failures[ip]
	if !ok {
		return 0
	}
	if now-ft.firstSeen > int64(s.window) {
		delete(s.failures, ip)
		return 0
	}
	return ft.count
}

// ReputationResult holds the computed reputation score and breakdown.
type ReputationResult struct {
	Score         int
	Action        string // "allow", "challenge", "throttle", "block"
	Breakdown     map[string]int
	IsTorExit     bool
	IsVPN         bool
	IsDataCenter  bool
	IsThreatListed bool
	FailureCount  int
}

// NewIPReputationScorer creates a scorer with all signal providers.
func NewIPReputationScorer(
	tor *TorExitChecker,
	vpn *VPNChecker,
	geo *GeoAnomalyDetector,
	failures *AuthFailureStore,
	feeds ThreatFeedProvider,
) *IPReputationScorer {
	return &IPReputationScorer{
		torChecker:   tor,
		vpnChecker:   vpn,
		geoDetector:  geo,
		failureStore: failures,
		threatFeeds:  feeds,
	}
}

// Score computes the reputation score for an IP.
func (s *IPReputationScorer) Score(ip string, now int64) *ReputationResult {
	result := &ReputationResult{
		Breakdown: make(map[string]int),
	}

	// 1. Threat intel feeds (weight: 35)
	if s.threatFeeds != nil && s.threatFeeds.IsListed(ip) {
		feedScore := s.threatFeeds.GetScore(ip)
		contrib := minInt(feedScore*35/100, 35)
		result.Score += contrib
		result.Breakdown["threat_feed"] = contrib
		result.IsThreatListed = true
	}

	// 2. Tor exit node (weight: 25)
	if s.torChecker != nil && s.torChecker.IsTorExit(ip) {
		result.Score += 25
		result.Breakdown["tor"] = 25
		result.IsTorExit = true
	}

	// 3. VPN / data center (weight: 20)
	if s.vpnChecker != nil {
		vpnResult := s.vpnChecker.Check(ip)
		if vpnResult.IsDataCenter {
			result.Score += 20
			result.Breakdown["data_center"] = 20
			result.IsDataCenter = true
		} else if vpnResult.IsVPN {
			result.Score += 15
			result.Breakdown["vpn"] = 15
			result.IsVPN = true
		}
	}

	// 4. Auth failure history (weight: 20, scaled by count)
	if s.failureStore != nil {
		failCount := s.failureStore.GetFailureCount(ip, now)
		result.FailureCount = failCount
		if failCount > 0 {
			failContrib := minInt(failCount*4, 20) // 5 failures = 20 points
			result.Score += failContrib
			result.Breakdown["auth_failures"] = failContrib
		}
	}

	// Cap at 100
	result.Score = minInt(result.Score, 100)

	// Determine action
	switch {
	case result.Score >= 80:
		result.Action = "block"
	case result.Score >= 60:
		result.Action = "throttle"
	case result.Score >= 30:
		result.Action = "challenge"
	default:
		result.Action = "allow"
	}

	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

---

## 6. fail2ban Integration Patterns

### 6.1 Classic fail2ban

fail2ban is a host-based intrusion prevention tool that monitors log files for authentication failures and bans IPs via iptables/firewalld. The standard workflow:

1. **Monitor** — fail2ban tails `/var/log/auth.log` (SSH) or custom log files.
2. **Match** — Regex patterns identify failed login attempts.
3. **Ban** — After N failures in M seconds, ban the IP for T seconds.

For IAM, the equivalent is: monitor auth failures in NATS/audit events and ban IPs at the gateway level.

### 6.2 GGID Equivalent: NATS-Driven IP Ban List

GGID's architecture already publishes auth events to NATS JetStream. The audit service publishes events like:

```json
{
  "event_type": "auth.login.failed",
  "ip": "203.0.113.50",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2025-01-15T14:30:00Z"
}
```

A NATS subscriber can aggregate these failures and auto-ban IPs that exceed thresholds.

### 6.3 Go Implementation

```go
package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// NATSBanManager subscribes to auth failure events and auto-bans IPs.
type NATSBanManager struct {
	mu            sync.Mutex
	failures      map[string]*ipFailTracker // ip → failure tracker
	banned        map[string]*banEntry       // ip → ban entry
	maxFailures   int                        // failures before ban
	banDuration   time.Duration
	window        time.Duration // failure counting window
	banListUpdater func(ip string, reason string, ttl time.Duration) // callback to update gateway
}

type ipFailTracker struct {
	count    int
	firstHit time.Time
	lastHit  time.Time
}

type banEntry struct {
	IP        string
	Reason    string
	BannedAt  time.Time
	ExpiresAt time.Time
}

// AuthFailureEvent represents an auth failure event from NATS.
type AuthFailureEvent struct {
	EventType string `json:"event_type"`
	IP        string `json:"ip"`
	TenantID  string `json:"tenant_id"`
	Timestamp string `json:"timestamp"`
}

// NewNATSBanManager creates a ban manager.
// maxFailures: failures within window before banning.
// banDuration: how long the ban lasts.
func NewNATSBanManager(maxFailures int, window, banDuration time.Duration) *NATSBanManager {
	if maxFailures <= 0 {
		maxFailures = 10
	}
	if window <= 0 {
		window = 10 * time.Minute
	}
	if banDuration <= 0 {
		banDuration = 1 * time.Hour
	}
	return &NATSBanManager{
		failures:    make(map[string]*ipFailTracker),
		banned:      make(map[string]*banEntry),
		maxFailures: maxFailures,
		window:      window,
		banDuration: banDuration,
	}
}

// SetBanListUpdater sets the callback invoked when an IP is banned.
// The gateway registers this to update its IP filter store.
func (m *NATSBanManager) SetBanListUpdater(fn func(ip, reason string, ttl time.Duration)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.banListUpdater = fn
}

// ProcessEvent handles an auth failure event from NATS.
func (m *NATSBanManager) ProcessEvent(data []byte) {
	var event AuthFailureEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("ban manager: failed to parse event: %v", err)
		return
	}
	if event.IP == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already banned
	if _, banned := m.banned[event.IP]; banned {
		return
	}

	now := time.Now()
	tracker, ok := m.failures[event.IP]
	if !ok || now.Sub(tracker.firstHit) > m.window {
		m.failures[event.IP] = &ipFailTracker{
			count:    1,
			firstHit: now,
			lastHit:  now,
		}
		return
	}

	tracker.count++
	tracker.lastHit = now

	// Check if ban threshold reached
	if tracker.count >= m.maxFailures {
		reason := fmt.Sprintf("auto-ban: %d auth failures in %v", tracker.count, m.window)
		entry := &banEntry{
			IP:        event.IP,
			Reason:    reason,
			BannedAt:  now,
			ExpiresAt: now.Add(m.banDuration),
		}
		m.banned[event.IP] = entry
		delete(m.failures, event.IP)

		// Notify gateway to update its IP filter
		if m.banListUpdater != nil {
			go m.banListUpdater(event.IP, reason, m.banDuration)
		}

		log.Printf("ban manager: banned %s (%s) until %s", event.IP, reason, entry.ExpiresAt)
	}
}

// IsBanned checks if an IP is currently banned.
func (m *NATSBanManager) IsBanned(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.banned[ip]
	if !ok {
		return false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(m.banned, ip)
		return false
	}
	return true
}

// CleanupExpired removes expired bans and stale failure trackers.
func (m *NATSBanManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for ip, entry := range m.banned {
		if now.After(entry.ExpiresAt) {
			delete(m.banned, ip)
		}
	}
	for ip, tracker := range m.failures {
		if now.Sub(tracker.firstHit) > m.window {
			delete(m.failures, ip)
		}
	}
}

// StartCleanup launches a periodic cleanup goroutine.
func (m *NATSBanManager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.CleanupExpired()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// BannedIPs returns a snapshot of currently banned IPs.
func (m *NATSBanManager) BannedIPs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ips := make([]string, 0, len(m.banned))
	for ip := range m.banned {
		ips = append(ips, ip)
	}
	return ips
}
```

---

## 7. Distributed IP Block List

### 7.1 Redis Design

For multi-instance GGID deployments (multiple gateway replicas), the IP block list must be shared across all instances. Redis provides the ideal data structure:

- **Sorted Set (`ZSET`)** — Score = expiry timestamp. Members are IP addresses.
- **Per-tenant namespacing** — Key format: `ipblock:{tenantID}` or global `ipblock:global`.
- **Auto-expiry** — Periodic cleanup removes entries past their expiry timestamp.

### 7.2 Go Implementation

```go
package threatintel

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisIPBlockList manages a distributed IP block list using Redis sorted sets.
type RedisIPBlockList struct {
	rdb         *redis.Client
	keyPrefix   string // e.g. "ipblock"
	defaultTTL  time.Duration
}

// NewRedisIPBlockList creates a distributed block list backed by Redis.
func NewRedisIPBlockList(rdb *redis.Client, defaultTTL time.Duration) *RedisIPBlockList {
	if defaultTTL <= 0 {
		defaultTTL = 1 * time.Hour
	}
	return &RedisIPBlockList{
		rdb:        rdb,
		keyPrefix:  "ipblock",
		defaultTTL: defaultTTL,
	}
}

// blockKey returns the Redis key for a tenant's block list.
func (b *RedisIPBlockList) blockKey(tenantID string) string {
	if tenantID == "" {
		return b.keyPrefix + ":global"
	}
	return fmt.Sprintf("%s:%s", b.keyPrefix, tenantID)
}

// reasonKey returns the Redis key for a tenant's ban reasons.
func (b *RedisIPBlockList) reasonKey(tenantID string) string {
	return b.blockKey(tenantID) + ":reasons"
}

// Block adds an IP to the block list with a TTL and reason.
// tenantID can be empty for a global ban.
func (b *RedisIPBlockList) Block(
	ctx context.Context, tenantID, ip, reason string, ttl time.Duration,
) error {
	if ttl <= 0 {
		ttl = b.defaultTTL
	}
	expiry := float64(time.Now().Add(ttl).Unix())

	pipe := b.rdb.TxPipeline()
	// Add IP to sorted set with expiry as score
	pipe.ZAdd(ctx, b.blockKey(tenantID), redis.Z{Score: expiry, Member: ip})
	// Store reason in a hash
	pipe.HSet(ctx, b.reasonKey(tenantID), ip, reason)
	// Set TTL on the reason hash to auto-expire
	pipe.Expire(ctx, b.reasonKey(tenantID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// IsBlocked checks if an IP is currently blocked for a tenant.
// Checks both tenant-specific and global block lists.
func (b *RedisIPBlockList) IsBlocked(ctx context.Context, tenantID, ip string) (bool, string, error) {
	now := float64(time.Now().Unix())

	// Check tenant-specific block list
	score, err := b.rdb.ZScore(ctx, b.blockKey(tenantID), ip).Result()
	if err == nil && score > now {
		reason, _ := b.rdb.HGet(ctx, b.reasonKey(tenantID), ip).Result()
		return true, reason, nil
	}

	// Check global block list (if tenantID is not empty)
	if tenantID != "" {
		score, err := b.rdb.ZScore(ctx, b.blockKey(""), ip).Result()
		if err == nil && score > now {
			reason, _ := b.rdb.HGet(ctx, b.reasonKey(""), ip).Result()
			return true, reason, nil
		}
	}

	return false, "", nil
}

// Unblock removes an IP from the block list.
func (b *RedisIPBlockList) Unblock(ctx context.Context, tenantID, ip string) error {
	pipe := b.rdb.TxPipeline()
	pipe.ZRem(ctx, b.blockKey(tenantID), ip)
	pipe.HDel(ctx, b.reasonKey(tenantID), ip)
	_, err := pipe.Exec(ctx)
	return err
}

// ListBlocked returns all currently blocked IPs for a tenant.
func (b *RedisIPBlockList) ListBlocked(ctx context.Context, tenantID string) ([]BlockedIP, error) {
	now := float64(time.Now().Unix())

	// Get all IPs with score > now (still active bans)
	members, err := b.rdb.ZRangeByScore(ctx, b.blockKey(tenantID), &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", now),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	results := make([]BlockedIP, 0, len(members))
	for _, ip := range members {
		score, _ := b.rdb.ZScore(ctx, b.blockKey(tenantID), ip).Result()
		reason, _ := b.rdb.HGet(ctx, b.reasonKey(tenantID), ip).Result()
		results = append(results, BlockedIP{
			IP:       ip,
			Reason:   reason,
			ExpiresAt: time.Unix(int64(score), 0),
		})
	}
	return results, nil
}

// BlockedIP represents a blocked IP entry.
type BlockedIP struct {
	IP        string
	Reason    string
	ExpiresAt time.Time
}

// CleanupExpired removes expired entries from the block list.
// This should run periodically (every 5 minutes) as a background task.
func (b *RedisIPBlockList) CleanupExpired(ctx context.Context, tenantID string) error {
	now := float64(time.Now().Unix())
	// Remove all entries with score < now (expired)
	_, err := b.rdb.ZRemRangeByScore(ctx, b.blockKey(tenantID), "-inf", fmt.Sprintf("%f", now)).Result()
	return err
}

// StartCleanup launches a periodic cleanup goroutine.
func (b *RedisIPBlockList) StartCleanup(ctx context.Context, tenantIDs []string, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, tid := range tenantIDs {
					b.CleanupExpired(ctx, tid)
				}
				b.CleanupExpired(ctx, "") // global
			case <-ctx.Done():
				return
			}
		}
	}()
}
```

---

## 8. GGID Gateway IP Filter Design

### 8.1 Existing Capabilities

After reviewing `services/gateway/internal/middleware/`, the GGID gateway already has substantial IP-related infrastructure:

| File | Capability | Status |
|------|-----------|--------|
| `ip_filter.go` | Per-tenant IP allowlist/denylist (CIDR) | **Complete** — manual CIDR lists |
| `geoip.go` | Country allowlist/blocklist via MaxMind DB | **Partial** — `lookupCountry` is a stub |
| `botdetect.go` | User-Agent based bot detection + behavioral rate limiting | **Complete** |
| `token_bucket.go` | Per-tenant + IP token-bucket rate limiting | **Complete** |
| `adaptive_geo_dedup.go` | Adaptive rate limiting + request dedup + geo enrichment | **Complete** |
| `ratelimit.go` | Basic rate limiting middleware | **Complete** |
| `sliding_ratelimit.go` | Sliding window rate limiter | **Complete** |

### 8.2 What Exists (Strengths)

1. **`IPFilterStore`** (ip_filter.go) — Solid per-tenant CIDR allowlist/denylist with `parseCIDRList()` and `ipInList()` helpers. This is the natural integration point for auto-banning.

2. **`ClientIP()`** (token_bucket.go) — Correctly extracts client IP from `X-Forwarded-For`, `X-Real-IP`, or `RemoteAddr`.

3. **`GeoIPMiddleware`** (geoip.go) — Country allow/block infrastructure exists but `lookupCountry` is an empty stub. Wiring in MaxMind GeoLite2 would make this functional.

4. **`BehavioralBotDetect`** (botdetect.go) — Per-IP sliding window request counting with configurable thresholds.

5. **`GeoEnricher`** (adaptive_geo_dedup.go) — Prefix-based IP→country mapping with placeholder ranges.

### 8.3 What is Missing

| Gap | Severity | Description |
|-----|----------|-------------|
| Tor exit node blocking | **High** | No Tor detection at all |
| VPN/proxy/data-center detection | **High** | No ASN-based data center flagging |
| Threat intel feed integration | **High** | No AbuseIPDB/OTX lookup |
| Impossible travel detection | **Medium** | No per-user geo history tracking |
| Multi-signal IP reputation scoring | **High** | No unified risk scoring |
| NATS-driven auto-banning | **High** | No event-driven IP banning |
| Redis distributed block list | **Medium** | Block lists are in-memory only |
| MaxMind DB integration | **Medium** | `lookupCountry` is a stub |
| Auth failure tracking per IP | **Medium** | Rate limiter counts requests, not failures |

### 8.4 Proposed Middleware Architecture

```
Request Flow:
  PanicRecovery
    → CORS
      → RequestID
        → RequestLogger
          → TenantResolver
            → RateLimit (token bucket)
              → IPReputationMiddleware (NEW)
                → TorBlockMiddleware (NEW)
                  → GeoIPMiddleware (enhanced)
                    → JWTAuth
                      → Backend Handler
```

The **IPReputationMiddleware** should:

1. Compute a reputation score using all available signals.
2. For auth endpoints: block if score >= 80, challenge (MFA step-up) if 30-79.
3. For API endpoints: throttle if score >= 60, allow otherwise.
4. Log the score to the audit trail for visibility.

### 8.5 Proposed `ip_reputation.go` Middleware

```go
package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

// IPReputationConfig configures the IP reputation middleware.
type IPReputationConfig struct {
	ScoreThresholdBlock     int // default 80
	ScoreThresholdChallenge int // default 30
	AuthPaths               []string
	Enabled                 bool
}

// DefaultIPReputationConfig returns sensible defaults.
func DefaultIPReputationConfig() *IPReputationConfig {
	return &IPReputationConfig{
		ScoreThresholdBlock:     80,
		ScoreThresholdChallenge: 30,
		AuthPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/api/v1/auth/reset",
			"/api/v1/auth/refresh",
		},
		Enabled: true,
	}
}

// IPReputationMiddleware evaluates IP reputation and applies actions.
// The scorerFunc is injected by the caller, connecting to the IPReputationScorer.
func IPReputationMiddleware(
	cfg *IPReputationConfig,
	scorerFunc func(ip string, now int64) *IPReputationResult,
) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultIPReputationConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip health checks
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			ip := ClientIP(r)
			result := scorerFunc(ip, time.Now().Unix())

			// Check if this is an auth endpoint
			isAuthEndpoint := false
			for _, p := range cfg.AuthPaths {
				if strings.HasPrefix(r.URL.Path, p) {
					isAuthEndpoint = true
					break
				}
			}

			// Apply actions based on score and endpoint type
			if result.Score >= cfg.ScoreThresholdBlock {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-IP-Reputation-Score", strconv.Itoa(result.Score))
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]any{
					"error":           "ip_blocked",
					"message":         "Your IP address has been flagged as high-risk.",
					"reputation_score": result.Score,
				})
				return
			}

			// For auth endpoints, challenge if score is elevated
			if isAuthEndpoint && result.Score >= cfg.ScoreThresholdChallenge {
				// Signal downstream to require step-up MFA
				w.Header().Set("X-IP-Reputation-Score", strconv.Itoa(result.Score))
				w.Header().Set("X-Require-StepUp", "true")
			}

			// Set reputation score header for observability
			w.Header().Set("X-IP-Reputation-Score", strconv.Itoa(result.Score))

			next.ServeHTTP(w, r)
		})
	}
}
```

---

## 9. Gap Analysis & Recommendations

### 9.1 Current State Assessment

GGID's gateway middleware has strong **rate limiting** and **CIDR-based IP filtering** foundations, but lacks any form of **threat intelligence integration**. The system is reactive (manual block lists) rather than proactive (automated threat scoring). An attacker with a fresh botnet can brute-force credentials with near-impunity because:

- Per-IP rate limits are bypassed by IP rotation.
- No Tor exit node blocking.
- No data-center IP detection.
- No external threat feed integration.
- Auth failures are not fed back into IP blocking in real-time.

### 9.2 Action Items

#### Action 1: Implement Tor Exit Node Blocking

- **Scope:** Add `TorExitChecker` with hourly background refresh, wire into auth endpoint middleware.
- **Effort:** 1–2 days
- **Impact:** Eliminates the easiest anonymity vector for automated auth attacks.
- **Files:** New `tor_block.go`, modify `middleware.go` chain.

#### Action 2: Implement NATS-Driven Auto-Banning

- **Scope:** Subscribe to `auth.login.failed` events on NATS, aggregate per-IP failures, auto-ban IPs exceeding threshold via `IPFilterStore`.
- **Effort:** 2–3 days
- **Impact:** Catches brute-force and credential stuffing in real-time, closing the gap between rate limits and actual auth failures.
- **Files:** New `nats_ban_manager.go`, integrate with existing `IPFilterStore`.

#### Action 3: Add Multi-Signal IP Reputation Scoring

- **Scope:** Implement `IPReputationScorer` combining Tor + ASN + threat feeds + auth failures. Wire as gateway middleware with score-based actions (block/challenge/allow).
- **Effort:** 3–5 days
- **Impact:** Comprehensive threat mitigation. Reduces credential stuffing success rate by 80%+ based on industry benchmarks.
- **Files:** New `ip_reputation.go`, modify `middleware.go` chain.

#### Action 4: Implement Redis Distributed Block List

- **Scope:** Replace in-memory `IPFilterStore` with Redis-backed sorted set. Per-tenant namespacing. Auto-expiry for temporary bans.
- **Effort:** 2–3 days
- **Impact:** Block lists propagate across all gateway replicas instantly. Critical for multi-instance HA deployments.
- **Files:** New `redis_ip_block.go`, refactor `ip_filter.go` to use Redis backend.

#### Action 5: Wire MaxMind GeoLite2 into `lookupCountry`

- **Scope:** Download GeoLite2-City.mmdb (free tier), implement actual `lookupCountry()` using `oschwald/maxminddb-golang`. Enable impossible travel detection.
- **Effort:** 1 day
- **Impact:** Unlocks geo-based risk scoring and country-level access policies.
- **Files:** Modify `geoip.go`, add MaxMind DB dependency.

### 9.3 Priority Matrix

| Priority | Action | Effort | Risk Reduction |
|----------|--------|--------|----------------|
| P0 | Tor exit node blocking | Low | High |
| P0 | NATS-driven auto-banning | Medium | Critical |
| P1 | Multi-signal reputation scoring | Medium-High | Critical |
| P1 | Redis distributed block list | Medium | High |
| P2 | MaxMind GeoLite2 wiring | Low | Medium |

### 9.4 Long-Term Vision

- **Machine learning anomaly detection** — Train a model on the audit log to detect novel attack patterns (IP sequences, timing anomalies, user-agent fingerprints).
- **Shared threat intelligence** — Federation of threat data across GGID tenants. If tenant A flags an IP, tenant B benefits.
- **Device fingerprinting** — Supplement IP reputation with device fingerprints (FIDO2, browser fingerprinting) for stronger signal when IP is unreliable.
- **CAEP/RISC integration** — Feed IP reputation events into the Shared Signals Framework for cross-domain security event propagation.

---

## Conclusion

IP reputation is the foundation of IAM threat defense. GGID's existing rate limiting and CIDR filtering infrastructure provides a solid base, but the gap between "static block lists" and "dynamic threat scoring" is the difference between surviving and thriving under attack. The five action items above, prioritised by effort-to-impact ratio, would bring GGID to parity with commercial IAM platforms (Auth0, Okta, Keycloak) in IP-based threat mitigation.

The architectural beauty of GGID's middleware chain is that each new capability (Tor blocking, NATS banning, reputation scoring) can be added as an independent middleware layer without modifying existing code. The `ClientIP()` helper, `IPFilterStore`, and event-driven NATS architecture are already in place — the missing piece is the intelligence layer that sits on top.
