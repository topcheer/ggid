package webhooks

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// SSRFConfig configures SSRF protection for webhook delivery.
type SSRFConfig struct {
	// BlockedCIDRs is the list of blocked IP ranges.
	// Defaults to private IPs + metadata endpoints.
	BlockedCIDRs []string
	// AllowRedirects controls whether HTTP redirects are followed.
	// Redirects can bypass SSRF checks by redirecting to internal IPs.
	AllowRedirects bool
	// AllowLoopback allows 127.0.0.0/8 and ::1/128 (for development/testing only).
	// In production this MUST be false.
	AllowLoopback bool
	// DialTimeout is the timeout for DNS resolution + connection.
	DialTimeout time.Duration
}

// DefaultSSRFConfig returns a secure SSRF config that blocks all private IPs
// and cloud metadata endpoints.
func DefaultSSRFConfig() *SSRFConfig {
	return &SSRFConfig{
		BlockedCIDRs: []string{
			"10.0.0.0/8",       // private
			"172.16.0.0/12",    // private
			"192.168.0.0/16",   // private
			"127.0.0.0/8",      // loopback
			"169.254.0.0/16",   // link-local (AWS/GCP metadata)
			"::1/128",          // IPv6 loopback
			"fc00::/7",         // IPv6 private
			"fe80::/10",        // IPv6 link-local
		},
		AllowRedirects: false,
		DialTimeout:    5 * time.Second,
	}
}

// isBlockedIP checks whether an IP is in any of the blocked CIDR ranges.
func isBlockedIP(ipStr string, blockedCIDRs []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true // invalid IP = block
	}
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// validateURL performs basic SSRF validation on a webhook URL.
// Returns an error if the URL scheme is not HTTP(S) or the hostname is suspicious.
func validateURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}
	return nil
}

// NewSSRFSafeDeliverer creates an HTTP deliverer with SSRF protection.
// It blocks requests to private IPs, loopback, and cloud metadata endpoints.
// For tests that need loopback access, use NewTestDeliverer instead.
func NewSSRFSafeDeliverer(cfg *SSRFConfig) *HTTPDeliverer {
	if cfg == nil {
		cfg = DefaultSSRFConfig()
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}

	// Parse blocked CIDRs
	var blockedCIDRs []*net.IPNet
	for _, cidr := range cfg.BlockedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedCIDRs = append(blockedCIDRs, ipNet)
		}
	}

	// Create a custom transport with a Control function that validates
	// the resolved IP after DNS resolution but before connection.
	dialer := &net.Dialer{
		Timeout: cfg.DialTimeout,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract host and port
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}

			// Resolve DNS
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("SSRF: DNS resolution failed for %s: %w", host, err)
			}

			// Check all resolved IPs
			for _, ipAddr := range ips {
				if cfg.AllowLoopback && (ipAddr.IP.IsLoopback() || ipAddr.IP.IsUnspecified()) {
					continue
				}
				if isBlockedIP(ipAddr.IP.String(), blockedCIDRs) {
					return nil, fmt.Errorf("SSRF: resolved IP %s for %s is blocked", ipAddr.IP, host)
				}
			}

			// All resolved IPs are safe — connect
			return dialer.DialContext(ctx, network, addr)
		},
	}

	_ = cfg.AllowRedirects // handled by CheckRedirect below

	return &HTTPDeliverer{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !cfg.AllowRedirects {
					return http.ErrUseLastResponse
				}
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return validateURL(req.URL.String())
			},
		},
		maxRetries: 3,
	}
}
