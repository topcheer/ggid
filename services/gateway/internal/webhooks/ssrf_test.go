package webhooks

import (
	"net"
	"testing"
)

func TestIsBlockedIP_Private_C22(t *testing.T) {
	cfg := DefaultSSRFConfig()
	var cidrs []*net.IPNet
	for _, c := range cfg.BlockedCIDRs {
		_, ipNet, _ := net.ParseCIDR(c)
		cidrs = append(cidrs, ipNet)
	}

	blocked := []string{
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.1.1",
		"192.168.0.100",
		"127.0.0.1",
		"127.0.0.1",
		"169.254.169.254", // AWS metadata
		"::1",
		"fc00::1",
		"fe80::1",
	}
	for _, ip := range blocked {
		if !isBlockedIP(ip, cidrs) {
			t.Errorf("%s should be blocked", ip)
		}
	}
}

func TestIsBlockedIP_Public_C22(t *testing.T) {
	cfg := DefaultSSRFConfig()
	var cidrs []*net.IPNet
	for _, c := range cfg.BlockedCIDRs {
		_, ipNet, _ := net.ParseCIDR(c)
		cidrs = append(cidrs, ipNet)
	}

	public := []string{
		"8.8.8.8",
		"1.1.1.1",
		"203.0.113.1",
		"2001:4860:4860::8888",
	}
	for _, ip := range public {
		if isBlockedIP(ip, cidrs) {
			t.Errorf("%s should NOT be blocked", ip)
		}
	}
}

func TestIsBlockedIP_Invalid_C22(t *testing.T) {
	// Invalid IP should be blocked
	if !isBlockedIP("not-an-ip", nil) {
		t.Error("invalid IP should be blocked")
	}
}

func TestValidateURL_NonHTTP_C22(t *testing.T) {
	if err := validateURL("file:///etc/passwd"); err == nil {
		t.Error("file:// should be rejected")
	}
	if err := validateURL("ftp://example.com/file"); err == nil {
		t.Error("ftp:// should be rejected")
	}
	if err := validateURL("gopher://localhost:6379/flushall"); err == nil {
		t.Error("gopher:// should be rejected")
	}
}

func TestValidateURL_Valid_C22(t *testing.T) {
	if err := validateURL("https://example.com/webhook"); err != nil {
		t.Errorf("valid HTTPS URL rejected: %v", err)
	}
	if err := validateURL("http://example.com/webhook"); err != nil {
		t.Errorf("valid HTTP URL rejected: %v", err)
	}
}

func TestNewSSRFSafeDeliverer_C22(t *testing.T) {
	d := NewSSRFSafeDeliverer(nil) // uses default config
	if d == nil {
		t.Fatal("nil deliverer")
	}
	if d.client == nil {
		t.Error("client should not be nil")
	}
	if d.maxRetries != 3 {
		t.Errorf("maxRetries=%d want 3", d.maxRetries)
	}
}

func TestDefaultSSRFConfig_C22(t *testing.T) {
	cfg := DefaultSSRFConfig()
	if len(cfg.BlockedCIDRs) < 7 {
		t.Errorf("expected >=7 blocked CIDRs, got %d", len(cfg.BlockedCIDRs))
	}
	if cfg.AllowRedirects {
		t.Error("redirects should be blocked by default")
	}
}
