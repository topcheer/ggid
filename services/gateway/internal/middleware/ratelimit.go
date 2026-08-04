package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig defines per-endpoint rate limits.
type RateLimitConfig struct {
	LoginLimit    int           // requests per minute for login
	RegisterLimit int           // requests per minute for register
	TokenLimit    int           // requests per minute for /oauth/token (brute-force protection)
	APILimit      int           // requests per minute for general API
	Window        time.Duration // sliding window
}

// DefaultRateLimitConfig returns production-ready defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		LoginLimit:    5,
		RegisterLimit: 3,
		TokenLimit:    10, // strict limit to prevent token brute-force
		APILimit:      100,
		Window:        time.Minute,
	}
}

// RateLimiter provides in-memory fixed-window rate limiting.
// For production with multiple gateway instances, replace with Redis-backed limiter.
type rateBucket struct {
	count    int
	expireAt time.Time
}

type RateLimiter struct {
	cfg       RateLimitConfig
	mu        sync.Mutex
	buckets   map[string]*rateBucket
	done      chan struct{}
	closeOnce sync.Once
}

// NewRateLimiter creates a new rate limiter with the given config.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		cfg:     cfg,
		buckets: make(map[string]*rateBucket),
		done:    make(chan struct{}),
	}
	// Background cleanup of expired buckets
	go rl.cleanup()
	return rl
}

// StopCleanup terminates the background cleanup goroutine.
func (rl *RateLimiter) StopCleanup() {
	rl.closeOnce.Do(func() { close(rl.done) })
}

// Middleware returns an HTTP middleware that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for non-API paths
		if r.URL.Path == "/healthz" || r.URL.Path == "/docs" ||
			r.URL.Path == "/api-docs" || r.URL.Path == "/login" ||
			r.URL.Path == "/register" {
			next.ServeHTTP(w, r)
			return
		}

		limit := rl.getLimit(r.URL.Path)
		if limit == 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := rl.bucketKey(r)
		allowed, remaining, resetAt := rl.allow(key, limit)

		// Set standard rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			secs := int(time.Until(resetAt).Seconds())
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimit(path string) int {
	switch {
	case path == "/api/v1/auth/verify" || path == "/api/v1/auth/login":
		return rl.cfg.LoginLimit
	case path == "/api/v1/auth/register":
		return rl.cfg.RegisterLimit
	case path == "/oauth/token":
		return rl.cfg.TokenLimit
	case len(path) > 8 && path[:8] == "/api/v1/":
		return rl.cfg.APILimit
	default:
		// SECURITY: Apply a default rate limit to all unmapped paths
		// (SCIM, SAML, OAuth non-token) instead of unlimited (0).
		if rl.cfg.APILimit > 0 {
			return rl.cfg.APILimit
		}
		return 600 // default: 600 req/min per IP
	}
}

func (rl *RateLimiter) bucketKey(r *http.Request) string {
	ip := clientIPFromRequest(r)
	return fmt.Sprintf("%s:%s", r.URL.Path, ip)
}

// clientIPFromRequest extracts the real client IP safely.
// SECURITY: Only trusts X-Forwarded-For from the rightmost entry when the
// direct connection is from a trusted proxy. Without a trusted proxy,
// falls back to TCP RemoteAddr (ground truth, not spoofable by clients).
func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if isTrustedProxy(host) {
		// X-Forwarded-For: client, proxy1, proxy2 — take the LAST entry
		// (appended by our trusted ingress proxy).
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			last := strings.TrimSpace(parts[len(parts)-1])
			if last != "" {
				return last
			}
		}
	}

	return host
}

// trustedProxyCIDRs holds IPs/CIDRs allowed to set X-Forwarded-For.
// Configured via GGID_TRUSTED_PROXIES (comma-separated). Defaults to
// loopback and private ranges (common single-proxy deployment).
var trustedProxies = func() map[string]bool {
	m := map[string]bool{
		"127.0.0.1": true, "::1": true,
	}
	if extra := os.Getenv("GGID_TRUSTED_PROXIES"); extra != "" {
		for _, ip := range strings.Split(extra, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				m[ip] = true
			}
		}
	}
	return m
}()

func isTrustedProxy(host string) bool {
	if trustedProxies[host] {
		return true
	}
	// Check private network prefixes (10.x, 172.16-31.x, 192.168.x)
	if strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") {
		return true
	}
	if strings.HasPrefix(host, "172.") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 {
			octet, _ := strconv.Atoi(parts[1])
			if octet >= 16 && octet <= 31 {
				return true
			}
		}
	}
	return false
}

// isTrustedProxyHost checks if a RemoteAddr (host:port) is from a trusted proxy.
func isTrustedProxyHost(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return isTrustedProxy(host)
}

func (rl *RateLimiter) allow(key string, limit int) (allowed bool, remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]

	if !exists || now.After(bucket.expireAt) {
		// New window
		resetAt = now.Add(rl.cfg.Window)
		rl.buckets[key] = &rateBucket{count: 1, expireAt: resetAt}
		return true, limit - 1, resetAt
	}

	if bucket.count >= limit {
		return false, 0, bucket.expireAt
	}

	bucket.count++
	return true, limit - bucket.count, bucket.expireAt
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.buckets {
				if now.After(b.expireAt) {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		}
	}
}
