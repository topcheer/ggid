// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
)

// CanaryConfig defines a per-route canary deployment.
type CanaryConfig struct {
	StableURL    string // primary backend URL
	CanaryURL    string // canary backend URL
	Percentage   int    // 0–100, percent of traffic to the canary
	Header       string // optional header name to force canary (e.g. "X-Canary")
	CookieName   string // optional cookie name for sticky canary
}

// CanaryRouter routes a percentage of requests to the canary backend.
// The decision is deterministic per-request: header > cookie > random.
type CanaryRouter struct {
	configs map[string]*CanaryConfig // route prefix → config
	counter atomic.Uint64            // round-robin counter for deterministic distribution
}

// NewCanaryRouter creates a canary router from the given configs.
func NewCanaryRouter(configs map[string]*CanaryConfig) *CanaryRouter {
	return &CanaryRouter{configs: configs}
}

// ShouldRouteCanary decides whether the request should go to the canary
// backend. It evaluates in order: header override, sticky cookie, then
// percentage-based random / counter.
func (cr *CanaryRouter) ShouldRouteCanary(cfg *CanaryConfig, r *http.Request) bool {
	// 1. Header override — always canary if header present
	if cfg.Header != "" {
		if h := r.Header.Get(cfg.Header); h != "" {
			if h == "true" || h == "1" {
				return true
			}
			if h == "false" || h == "0" {
				return false
			}
		}
	}

	// 2. Sticky cookie — if present, use its value
	if cfg.CookieName != "" {
		if c, err := r.Cookie(cfg.CookieName); err == nil {
			if c.Value == "canary" {
				return true
			}
			if c.Value == "stable" {
				return false
			}
		}
	}

	// 3. Percentage-based routing
	if cfg.Percentage <= 0 {
		return false
	}
	if cfg.Percentage >= 100 {
		return true
	}
	// Use counter for deterministic distribution
	n := cr.counter.Add(1)
	return int(n%100) < cfg.Percentage
}

// SetCanaryCookie writes a sticky cookie so the client stays on the
// same backend for subsequent requests.
func SetCanaryCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetCanaryConfig returns the canary config for the given route prefix, or nil.
func (cr *CanaryRouter) GetCanaryConfig(prefix string) *CanaryConfig {
	if cr == nil {
		return nil
	}
	return cr.configs[prefix]
}

// --- Deterministic test helpers ---

// pickByPercentage is a pure function for testability.
func pickByPercentage(percentage int, n uint64) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}
	return int(n%100) < percentage
}

// RandomBool is kept for tests that need non-deterministic coverage.
func RandomBool(percentage int) bool {
	return rand.Intn(100) < percentage
}

// ParsePercentage parses an integer percentage from a string (0-100).
func ParsePercentage(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
