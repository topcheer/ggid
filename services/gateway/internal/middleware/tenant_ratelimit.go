package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TenantRateLimitConfig holds per-tenant rate limit settings.
type TenantRateLimitConfig struct {
	TenantID   string `json:"tenant_id"`
	RequestsPerMin  int   `json:"requests_per_min"`
	BurstSize       int   `json:"burst_size"`
	Enabled         bool  `json:"enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TenantRateLimitStore manages per-tenant rate limit configs.
// In production, this uses Redis. For testing, it uses in-memory map.
type TenantRateLimitStore struct {
	mu      sync.RWMutex
	configs map[string]*TenantRateLimitConfig
	defaultConfig TenantRateLimitConfig
}

// NewTenantRateLimitStore creates a new store with the given default config.
func NewTenantRateLimitStore(defaultReq, defaultBurst int) *TenantRateLimitStore {
	return &TenantRateLimitStore{
		configs: make(map[string]*TenantRateLimitConfig),
		defaultConfig: TenantRateLimitConfig{
			RequestsPerMin: defaultReq,
			BurstSize:      defaultBurst,
			Enabled:        true,
		},
	}
}

// Get returns the rate limit config for a tenant, falling back to defaults.
func (s *TenantRateLimitStore) Get(tenantID string) TenantRateLimitConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cfg, ok := s.configs[tenantID]; ok {
		return *cfg
	}
	return TenantRateLimitConfig{
		TenantID:       tenantID,
		RequestsPerMin: s.defaultConfig.RequestsPerMin,
		BurstSize:      s.defaultConfig.BurstSize,
		Enabled:        s.defaultConfig.Enabled,
	}
}

// Set updates the rate limit config for a tenant.
func (s *TenantRateLimitStore) Set(cfg TenantRateLimitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg.UpdatedAt = time.Now()
	s.configs[cfg.TenantID] = &cfg
}

// Delete removes a tenant's custom config, reverting to defaults.
func (s *TenantRateLimitStore) Delete(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, tenantID)
}

// List returns all configured tenant rate limits.
func (s *TenantRateLimitStore) List() []TenantRateLimitConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]TenantRateLimitConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		result = append(result, *cfg)
	}
	return result
}

// TenantRateLimitHandler returns HTTP handlers for per-tenant rate limit management.
func TenantRateLimitHandler(store *TenantRateLimitStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant_id from path: /api/v1/gateway/ratelimits/{tenant_id}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/gateway/ratelimits"), "/")
		var tenantID string
		if len(parts) >= 1 && parts[0] != "" {
			tenantID = parts[0]
		}

		switch r.Method {
		case http.MethodGet:
			if tenantID == "" {
				// List all
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"configs": store.List(),
					"default": store.defaultConfig,
				})
				return
			}
			cfg := store.Get(tenantID)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)

		case http.MethodPut:
			var cfg TenantRateLimitConfig
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
				return
			}
			if tenantID != "" {
				cfg.TenantID = tenantID
			}
			if cfg.RequestsPerMin <= 0 {
				cfg.RequestsPerMin = store.defaultConfig.RequestsPerMin
			}
			if cfg.BurstSize <= 0 {
				cfg.BurstSize = store.defaultConfig.BurstSize
			}
			store.Set(cfg)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cfg)

		case http.MethodDelete:
			if tenantID == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			store.Delete(tenantID)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
