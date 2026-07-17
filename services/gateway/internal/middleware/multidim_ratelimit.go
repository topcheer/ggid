package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Dimension identifies a rate limiting dimension.
type Dimension string

const (
	DimTenant  Dimension = "tenant"
	DimUser    Dimension = "user"
	DimAPIKey  Dimension = "api_key"
	DimIP      Dimension = "ip"
	DimEndpoint Dimension = "endpoint"
)

// MultiDimRateLimit defines burst + sustained limits for a dimension.
type MultiDimRateLimit struct {
	BurstPerMin      int // short window burst limit
	SustainedPerHour int // long window sustained limit (0 = unlimited)
}

// MultiDimTierConfig defines per-tier limits across all 5 dimensions.
type MultiDimTierConfig struct {
	Tenant   MultiDimRateLimit
	User     MultiDimRateLimit
	APIKey   MultiDimRateLimit
	IP       MultiDimRateLimit
	Endpoint MultiDimRateLimit
}

// DefaultMultiDimConfigs returns standard configs for free/pro/enterprise tiers.
func DefaultMultiDimConfigs() map[Tier]MultiDimTierConfig {
	return map[Tier]MultiDimTierConfig{
		TierFree: {
			Tenant:   MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 1000},
			User:     MultiDimRateLimit{BurstPerMin: 30, SustainedPerHour: 300},
			APIKey:   MultiDimRateLimit{BurstPerMin: 60, SustainedPerHour: 600},
			IP:       MultiDimRateLimit{BurstPerMin: 200, SustainedPerHour: 2000},
			Endpoint: MultiDimRateLimit{BurstPerMin: 50, SustainedPerHour: 500},
		},
		TierPro: {
			Tenant:   MultiDimRateLimit{BurstPerMin: 500, SustainedPerHour: 10000},
			User:     MultiDimRateLimit{BurstPerMin: 100, SustainedPerHour: 2000},
			APIKey:   MultiDimRateLimit{BurstPerMin: 300, SustainedPerHour: 5000},
			IP:       MultiDimRateLimit{BurstPerMin: 1000, SustainedPerHour: 20000},
			Endpoint: MultiDimRateLimit{BurstPerMin: 200, SustainedPerHour: 5000},
		},
		TierEnterprise: {
			Tenant:   MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
			User:     MultiDimRateLimit{BurstPerMin: 500, SustainedPerHour: 10000},
			APIKey:   MultiDimRateLimit{BurstPerMin: 1000, SustainedPerHour: 50000},
			IP:       MultiDimRateLimit{BurstPerMin: 0, SustainedPerHour: 0},
			Endpoint: MultiDimRateLimit{BurstPerMin: 500, SustainedPerHour: 20000},
		},
	}
}

// dimBucket tracks burst (1-min) and sustained (1-hour) counters.
type dimBucket struct {
	burstCount    int
	burstExpireAt time.Time
	sustainedCount int
	sustainedExpireAt time.Time
}

// MultiDimRateLimiter enforces 5-dimensional rate limits with burst + sustained windows.
type MultiDimRateLimiter struct {
	mu      sync.Mutex
	tiers   map[Tier]MultiDimTierConfig
	buckets map[string]*dimBucket // key: dimension:value
}

// NewMultiDimRateLimiter creates a new 5-dimensional rate limiter.
func NewMultiDimRateLimiter(configs map[Tier]MultiDimTierConfig) *MultiDimRateLimiter {
	if configs == nil {
		configs = DefaultMultiDimConfigs()
	}
	rl := &MultiDimRateLimiter{
		tiers:   configs,
		buckets: make(map[string]*dimBucket),
	}
	go rl.cleanup()
	return rl
}

// UpdateTier updates the limits for a specific tier.
func (rl *MultiDimRateLimiter) UpdateTier(tier Tier, cfg MultiDimTierConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tiers[tier] = cfg
}

// GetTier returns the config for a tier.
func (rl *MultiDimRateLimiter) GetTier(tier Tier) MultiDimTierConfig {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if cfg, ok := rl.tiers[tier]; ok {
		return cfg
	}
	return MultiDimTierConfig{}
}

// AllTiers returns all configured tiers.
func (rl *MultiDimRateLimiter) AllTiers() map[Tier]MultiDimTierConfig {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	result := make(map[Tier]MultiDimTierConfig, len(rl.tiers))
	for k, v := range rl.tiers {
		result[k] = v
	}
	return result
}

// RateLimitResult describes the outcome of a rate limit check.
type RateLimitResult struct {
	Allowed   bool
	Dimension Dimension
	Limit     int
	Remaining int
	ResetAt   time.Time
}

// Check evaluates all 5 dimensions and returns the most restrictive result.
func (rl *MultiDimRateLimiter) Check(tier Tier, tenantID, userID, apiKey, ip, endpoint string) RateLimitResult {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cfg, ok := rl.tiers[tier]
	if !ok {
		return RateLimitResult{Allowed: true} // unknown tier = allow
	}

	now := time.Now()
	checks := []struct {
		dim   Dimension
		val   string
		limit MultiDimRateLimit
	}{
		{DimTenant, tenantID, cfg.Tenant},
		{DimUser, userID, cfg.User},
		{DimAPIKey, apiKey, cfg.APIKey},
		{DimIP, ip, cfg.IP},
		{DimEndpoint, endpoint, cfg.Endpoint},
	}

	mostRestrictive := RateLimitResult{Allowed: true}

	for _, c := range checks {
		if c.val == "" || (c.limit.BurstPerMin == 0 && c.limit.SustainedPerHour == 0) {
			continue // no limit or unlimited
		}
		key := fmt.Sprintf("%s:%s", c.dim, c.val)
		b := rl.getOrCreate(key, now)

		// Check burst (1-min window).
		if c.limit.BurstPerMin > 0 && b.burstCount >= c.limit.BurstPerMin {
			if mostRestrictive.Allowed || b.burstExpireAt.Before(mostRestrictive.ResetAt) {
				mostRestrictive = RateLimitResult{
					Allowed: false, Dimension: c.dim, Limit: c.limit.BurstPerMin,
					Remaining: 0, ResetAt: b.burstExpireAt,
				}
			}
			continue
		}

		// Check sustained (1-hour window).
		if c.limit.SustainedPerHour > 0 && b.sustainedCount >= c.limit.SustainedPerHour {
			if mostRestrictive.Allowed || b.sustainedExpireAt.Before(mostRestrictive.ResetAt) {
				mostRestrictive = RateLimitResult{
					Allowed: false, Dimension: c.dim, Limit: c.limit.SustainedPerHour,
					Remaining: 0, ResetAt: b.sustainedExpireAt,
				}
			}
			continue
		}

		// Track remaining for reporting.
		remaining := c.limit.BurstPerMin - b.burstCount - 1
		if c.limit.BurstPerMin == 0 {
			remaining = c.limit.SustainedPerHour - b.sustainedCount - 1
		}
		if mostRestrictive.Allowed && (mostRestrictive.Limit == 0 || remaining < mostRestrictive.Remaining) {
			mostRestrictive = RateLimitResult{
				Allowed: true, Dimension: c.dim,
				Limit: c.limit.BurstPerMin, Remaining: remaining,
				ResetAt: b.burstExpireAt,
			}
		}
	}

	// If allowed, increment all counters.
	if mostRestrictive.Allowed {
		for _, c := range checks {
			if c.val == "" {
				continue
			}
			key := fmt.Sprintf("%s:%s", c.dim, c.val)
			b := rl.getOrCreate(key, now)
			b.burstCount++
			b.sustainedCount++
		}
	}

	return mostRestrictive
}

func (rl *MultiDimRateLimiter) getOrCreate(key string, now time.Time) *dimBucket {
	b, ok := rl.buckets[key]
	if !ok {
		b = &dimBucket{
			burstExpireAt:    now.Add(time.Minute),
			sustainedExpireAt: now.Add(time.Hour),
		}
		rl.buckets[key] = b
	}
	// Reset expired counters.
	if now.After(b.burstExpireAt) {
		b.burstCount = 0
		b.burstExpireAt = now.Add(time.Minute)
	}
	if now.After(b.sustainedExpireAt) {
		b.sustainedCount = 0
		b.sustainedExpireAt = now.Add(time.Hour)
	}
	return b
}

// GetUsage returns current usage stats for the caller's dimensions.
type DimensionUsage struct {
	Dimension     Dimension `json:"dimension"`
	Value         string    `json:"value"`
	BurstUsed     int       `json:"burst_used"`
	BurstLimit    int       `json:"burst_limit"`
	SustainedUsed int       `json:"sustained_used"`
	SustainedLimit int      `json:"sustained_limit"`
}

func (rl *MultiDimRateLimiter) GetUsage(tier Tier, tenantID, userID, apiKey, ip, endpoint string) []DimensionUsage {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cfg, ok := rl.tiers[tier]
	if !ok {
		return nil
	}
	now := time.Now()
	checks := []struct {
		dim   Dimension
		val   string
		limit MultiDimRateLimit
	}{
		{DimTenant, tenantID, cfg.Tenant},
		{DimUser, userID, cfg.User},
		{DimAPIKey, apiKey, cfg.APIKey},
		{DimIP, ip, cfg.IP},
		{DimEndpoint, endpoint, cfg.Endpoint},
	}

	var result []DimensionUsage
	for _, c := range checks {
		if c.val == "" {
			continue
		}
		key := fmt.Sprintf("%s:%s", c.dim, c.val)
		b, ok := rl.buckets[key]
		if !ok {
			b = &dimBucket{}
		}
		burstUsed := b.burstCount
		if now.After(b.burstExpireAt) {
			burstUsed = 0
		}
		sustainedUsed := b.sustainedCount
		if now.After(b.sustainedExpireAt) {
			sustainedUsed = 0
		}
		result = append(result, DimensionUsage{
			Dimension: c.dim, Value: c.val,
			BurstUsed: burstUsed, BurstLimit: c.limit.BurstPerMin,
			SustainedUsed: sustainedUsed, SustainedLimit: c.limit.SustainedPerHour,
		})
	}
	return result
}

func (rl *MultiDimRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.buckets {
			if now.After(b.burstExpireAt) && now.After(b.sustainedExpireAt) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// MultiDimRateLimitMiddleware checks all 5 dimensions, enforces most restrictive.
func MultiDimRateLimitMiddleware(limiter *MultiDimRateLimiter, tierResolver func(*http.Request) (Tier, string, string, string, string, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tier, tenantID, userID, apiKey, ip, endpoint := tierResolver(r)
			result := limiter.Check(tier, tenantID, userID, apiKey, ip, endpoint)

			// Set rate limit headers.
			w.Header().Set("X-RateLimit-Dimension", string(result.Dimension))
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
			w.Header().Set("X-RateLimit-Reset", result.ResetAt.Format(time.RFC3339))

			if !result.Allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(result.ResetAt).Seconds())+1))
				http.Error(w, fmt.Sprintf(`{"error":"rate_limit_exceeded","dimension":"%s","retry_after":"%s"}`,
					result.Dimension, result.ResetAt.Format(time.RFC3339)), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DefaultTierResolver extracts rate limit dimensions from request headers.
func DefaultTierResolver(r *http.Request) (Tier, string, string, string, string, string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	userID := r.Header.Get("X-User-ID")
	apiKey := r.Header.Get("X-API-Key")
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}
	endpoint := r.URL.Path

	tier := TierFree
	switch r.Header.Get("X-Tier") {
	case "pro":
		tier = TierPro
	case "enterprise":
		tier = TierEnterprise
	}
	return tier, tenantID, userID, apiKey, ip, endpoint
}
