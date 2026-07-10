package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// IPAllowlist checks if a client IP is in the allowed list.
type IPAllowlist struct {
	// CIDRs per tenant. Empty list = allow all (default open).
	tenantCIDRs map[string][]*net.IPNet
}

// NewIPAllowlist creates a new IP allowlist with the given tenant CIDRs.
func NewIPAllowlist(tenantCIDRs map[string][]*net.IPNet) *IPAllowlist {
	return &IPAllowlist{tenantCIDRs: tenantCIDRs}
}

// Middleware enforces IP restrictions. If a tenant has CIDRs configured,
// only requests from those IPs are allowed. No CIDRs = open access.
func (al *IPAllowlist) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := r.Context().Value(TenantIDKey).(string)
		if tenantID == "" {
			next.ServeHTTP(w, r)
			return
		}

		cidrs, ok := al.tenantCIDRs[tenantID]
		if !ok || len(cidrs) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := extractClientIP(r)
		ip := net.ParseIP(clientIP)
		if ip == nil {
			allowListDeny(w, "invalid client IP")
			return
		}

		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}

		allowListDeny(w, "IP not allowed")
	})
}

// extractClientIP gets the real client IP from headers or RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For (first IP)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr (strip port)
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// ParseCIDRs converts a list of CIDR strings to net.IPNet.
func ParseCIDRs(cidrs ...string) []*net.IPNet {
	var result []*net.IPNet
	for _, cidr := range cidrs {
		// Single IP without mask → /32 for IPv4
		if !strings.Contains(cidr, "/") {
			ip := net.ParseIP(cidr)
			if ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		result = append(result, ipNet)
	}
	return result
}

func allowListDeny(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
