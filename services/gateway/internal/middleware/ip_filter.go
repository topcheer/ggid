package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
)

// IPFilterConfig holds per-tenant IP allowlist/denylist rules.
type IPFilterConfig struct {
	AllowList []string // CIDR ranges to allow (if set, only these IPs can access)
	DenyList  []string // CIDR ranges to deny
	Enabled   bool
}

// IPFilterStore manages per-tenant IP filter configs.
type IPFilterStore struct {
	mu      sync.RWMutex
	configs map[string]*IPFilterConfig
	fallback *IPFilterConfig
}

// NewIPFilterStore creates a new store with a fallback config.
func NewIPFilterStore(fallback *IPFilterConfig) *IPFilterStore {
	return &IPFilterStore{
		configs:  make(map[string]*IPFilterConfig),
		fallback: fallback,
	}
}

// Set sets the IP filter config for a tenant.
func (s *IPFilterStore) Set(tenantID string, cfg *IPFilterConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[tenantID] = cfg
}

// Get returns the IP filter config for a tenant, or the fallback.
func (s *IPFilterStore) Get(tenantID string) *IPFilterConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cfg, ok := s.configs[tenantID]; ok {
		return cfg
	}
	return s.fallback
}

// Delete removes a per-tenant config.
func (s *IPFilterStore) Delete(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, tenantID)
}

// parseCIDRList parses a list of CIDR strings into networks.
func parseCIDRList(cidrs []string) []*net.IPNet {
	var networks []*net.IPNet
	for _, cidr := range cidrs {
		// Handle single IPs by appending /32 or /128
		if !strings.Contains(cidr, "/") {
			cidr = cidr + "/32"
		}
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			networks = append(networks, network)
		}
	}
	return networks
}

// ipInList checks if an IP is in any of the CIDR networks.
func ipInList(ip string, networks []*net.IPNet) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, network := range networks {
		if network.Contains(parsedIP) {
			return true
		}
	}
	return false
}

// IPFilterMiddleware enforces per-tenant IP allowlist/denylist.
func IPFilterMiddleware(store *IPFilterStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, _ := TenantIDFromRequest(r)
			cfg := store.Get(tenantID)

			// No config or disabled — allow all
			if cfg == nil || !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := ClientIP(r)

			// Check denylist first
			if len(cfg.DenyList) > 0 {
				denyNets := parseCIDRList(cfg.DenyList)
				if ipInList(clientIP, denyNets) {
					writeIPForbidden(w, "IP address blocked by denylist")
					return
				}
			}

			// Check allowlist
			if len(cfg.AllowList) > 0 {
				allowNets := parseCIDRList(cfg.AllowList)
				if !ipInList(clientIP, allowNets) {
					writeIPForbidden(w, "IP address not in allowlist")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeIPForbidden writes a 403 JSON error response.
func writeIPForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   "forbidden",
		"message": message,
	})
}
