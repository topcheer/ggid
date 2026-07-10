package middleware

import (
	"math"
	"net/http"
	"sync"
	"time"
)

// AdaptiveRateLimiter adjusts QPS limits based on backend response latency.
// When the backend is slow, it automatically lowers the limit. When fast,
// it gradually increases it. This protects backends from overload without
// manual tuning.
type AdaptiveRateLimiter struct {
	mu             sync.Mutex
	limits         map[string]*adaptiveBucket
	baseLimit      float64       // starting QPS
	minLimit       float64       // floor
	maxLimit       float64       // ceiling
	adjustInterval time.Duration // how often to adjust
	latencyThreshold time.Duration // above this → decrease
}

type adaptiveBucket struct {
	currentLimit float64
	avgLatency   time.Duration
	reqCount     int64
	latencySum   time.Duration
	lastAdjust   time.Time
	tokens       float64 // token bucket
	lastRefill   time.Time
}

// NewAdaptiveRateLimiter creates an adaptive limiter.
func NewAdaptiveRateLimiter(baseLimit float64, minLimit, maxLimit float64) *AdaptiveRateLimiter {
	if baseLimit <= 0 {
		baseLimit = 100
	}
	if minLimit <= 0 {
		minLimit = 10
	}
	if maxLimit <= 0 {
		maxLimit = 1000
	}
	return &AdaptiveRateLimiter{
		limits:          make(map[string]*adaptiveBucket),
		baseLimit:       baseLimit,
		minLimit:        minLimit,
		maxLimit:        maxLimit,
		adjustInterval:  10 * time.Second,
		latencyThreshold: 500 * time.Millisecond,
	}
}

// Allow checks if a request is allowed for the given key (e.g. tenant ID).
// Returns true if allowed, false if rate-limited.
func (al *AdaptiveRateLimiter) Allow(key string) bool {
	al.mu.Lock()
	defer al.mu.Unlock()

	b := al.getOrCreate(key)
	now := time.Now()

	// Refill tokens
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.currentLimit
	if b.tokens > b.currentLimit {
		b.tokens = b.currentLimit
	}
	b.lastRefill = now

	// Check if we have a token
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	b.reqCount++
	return true
}

// RecordLatency feeds backend response latency into the adaptive algorithm.
// Slow responses cause the limit to decrease; fast responses allow increase.
func (al *AdaptiveRateLimiter) RecordLatency(key string, latency time.Duration) {
	al.mu.Lock()
	defer al.mu.Unlock()

	b := al.getOrCreate(key)
	b.latencySum += latency

	// Adjust if enough time has passed
	if time.Since(b.lastAdjust) < al.adjustInterval {
		return
	}

	// Compute average latency
	if b.reqCount > 0 {
		b.avgLatency = b.latencySum / time.Duration(b.reqCount)
	}

	// Adjust limit based on latency
	if b.avgLatency > al.latencyThreshold {
		// Slow → decrease by 20%
		b.currentLimit *= 0.8
		if b.currentLimit < al.minLimit {
			b.currentLimit = al.minLimit
		}
	} else {
		// Fast → increase by 10%
		b.currentLimit *= 1.1
		if b.currentLimit > al.maxLimit {
			b.currentLimit = al.maxLimit
		}
	}

	// Reset counters
	b.reqCount = 0
	b.latencySum = 0
	b.lastAdjust = time.Now()
}

// Limit returns the current adaptive limit for a key.
func (al *AdaptiveRateLimiter) Limit(key string) float64 {
	al.mu.Lock()
	defer al.mu.Unlock()
	b, ok := al.limits[key]
	if !ok {
		return al.baseLimit
	}
	return math.Round(b.currentLimit)
}

// AllLimits returns all per-key limits.
func (al *AdaptiveRateLimiter) AllLimits() map[string]float64 {
	al.mu.Lock()
	defer al.mu.Unlock()
	result := make(map[string]float64, len(al.limits))
	for k, b := range al.limits {
		result[k] = math.Round(b.currentLimit)
	}
	return result
}

// SetLimit manually overrides the limit for a key.
func (al *AdaptiveRateLimiter) SetLimit(key string, limit float64) {
	al.mu.Lock()
	defer al.mu.Unlock()
	b := al.getOrCreate(key)
	if limit < al.minLimit {
		limit = al.minLimit
	}
	if limit > al.maxLimit {
		limit = al.maxLimit
	}
	b.currentLimit = limit
}

func (al *AdaptiveRateLimiter) getOrCreate(key string) *adaptiveBucket {
	b, ok := al.limits[key]
	if !ok {
		b = &adaptiveBucket{
			currentLimit: al.baseLimit,
			tokens:       al.baseLimit,
			lastRefill:   time.Now(),
			lastAdjust:   time.Now(),
		}
		al.limits[key] = b
	}
	return b
}

// --- Request Deduplication (Idempotency-Key) ---

// RequestDeduplicator caches responses by Idempotency-Key header.
// Only applies to safe methods (GET, HEAD). Uses in-memory TTL cache.
type RequestDeduplicator struct {
	mu      sync.RWMutex
	entries map[string]*dedupEntry
	ttl     time.Duration
}

type dedupEntry struct {
	statusCode int
	body       []byte
	headers    http.Header
	cachedAt   time.Time
}

// NewRequestDeduplicator creates a deduplicator with the given TTL.
func NewRequestDeduplicator(ttl time.Duration) *RequestDeduplicator {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &RequestDeduplicator{
		entries: make(map[string]*dedupEntry),
		ttl:     ttl,
	}
}

// Get returns a cached response for the given idempotency key, if valid.
func (d *RequestDeduplicator) Get(key string) (*dedupEntry, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.entries[key]
	if !ok || time.Since(entry.cachedAt) > d.ttl {
		return nil, false
	}
	return entry, true
}

// Set caches a response for the given idempotency key.
func (d *RequestDeduplicator) Set(key string, statusCode int, body []byte, headers http.Header) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries[key] = &dedupEntry{
		statusCode: statusCode,
		body:       body,
		headers:    headers.Clone(),
		cachedAt:   time.Now(),
	}
}

// DedupMiddleware serves cached responses for requests with matching
// Idempotency-Key headers (GET/HEAD only).
func DedupMiddleware(dedup *RequestDeduplicator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only dedup safe methods
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check cache
			if entry, ok := dedup.Get(key); ok {
				for k, v := range entry.headers {
					w.Header()[k] = v
				}
				w.Header().Set("X-Deduplicated", "true")
				w.WriteHeader(entry.statusCode)
				w.Write(entry.body)
				return
			}

			// Capture response and cache it
			rec := &dedupRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			if rec.status >= 200 && rec.status < 300 {
				dedup.Set(key, rec.status, rec.body, rec.Header())
			}
		})
	}
}

func (d *RequestDeduplicator) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.entries)
}

func (d *RequestDeduplicator) Delete(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.entries, key)
}

type dedupRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (r *dedupRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *dedupRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

// --- IP Geolocation Enrichment ---

// GeoEnricher enriches requests with geolocation headers based on client IP.
// It reads from a configurable IP-range → location mapping.
type GeoEnricher struct {
	ranges []geoIPRange
}

type geoIPRange struct {
	prefix  string // CIDR or prefix match
	country string
	city    string
}

// NewGeoEnricher creates a geo enricher with default mappings.
// In production, this would use a MaxMind GeoIP2 database.
func NewGeoEnricher() *GeoEnricher {
	return &GeoEnricher{
		ranges: []geoIPRange{
			{"10.", "internal", "datacenter"},
			{"192.168.", "internal", "lan"},
			{"172.16.", "internal", "lan"},
			{"127.", "local", "loopback"},
		},
	}
}

// AddRange registers a custom IP prefix → location mapping.
func (g *GeoEnricher) AddRange(prefix, country, city string) {
	g.ranges = append(g.ranges, geoIPRange{prefix, country, city})
}

// Lookup returns the country and city for a given IP address.
func (g *GeoEnricher) Lookup(ip string) (country, city string) {
	for _, r := range g.ranges {
		if len(ip) >= len(r.prefix) && ip[:len(r.prefix)] == r.prefix {
			return r.country, r.city
		}
	}
	return "unknown", ""
}

// GeoEnrichMiddleware injects X-Geo-Country and X-Geo-City headers
// based on the client IP address.
func GeoEnrichMiddleware(enricher *GeoEnricher) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only set if not already present (don't override upstream)
			if r.Header.Get("X-Geo-Country") == "" {
				country, city := enricher.Lookup(clientIP(r))
				r.Header.Set("X-Geo-Country", country)
				if city != "" {
					r.Header.Set("X-Geo-City", city)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from request, respecting X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
