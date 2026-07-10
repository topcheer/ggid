package transport

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d", cfg.MaxIdleConns)
	}
	if cfg.MaxConnsPerHost != 10 {
		t.Errorf("MaxConnsPerHost = %d", cfg.MaxConnsPerHost)
	}
	if cfg.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v", cfg.IdleConnTimeout)
	}
}

func TestLoadPoolConfigFromEnv(t *testing.T) {
	os.Setenv("GATEWAY_MAX_IDLE_CONNS", "200")
	os.Setenv("GATEWAY_MAX_CONNS_PER_HOST", "20")
	os.Setenv("GATEWAY_IDLE_TIMEOUT", "45")
	defer os.Unsetenv("GATEWAY_MAX_IDLE_CONNS")
	defer os.Unsetenv("GATEWAY_MAX_CONNS_PER_HOST")
	defer os.Unsetenv("GATEWAY_IDLE_TIMEOUT")

	cfg := LoadPoolConfigFromEnv()
	if cfg.MaxIdleConns != 200 {
		t.Errorf("MaxIdleConns = %d, want 200", cfg.MaxIdleConns)
	}
	if cfg.MaxConnsPerHost != 20 {
		t.Errorf("MaxConnsPerHost = %d, want 20", cfg.MaxConnsPerHost)
	}
	if cfg.IdleConnTimeout != 45*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 45s", cfg.IdleConnTimeout)
	}
}

func TestLoadPoolConfigFromEnv_InvalidValues(t *testing.T) {
	os.Setenv("GATEWAY_MAX_IDLE_CONNS", "not-a-number")
	os.Setenv("GATEWAY_MAX_CONNS_PER_HOST", "-5")
	os.Setenv("GATEWAY_IDLE_TIMEOUT", "0")
	defer os.Unsetenv("GATEWAY_MAX_IDLE_CONNS")
	defer os.Unsetenv("GATEWAY_MAX_CONNS_PER_HOST")
	defer os.Unsetenv("GATEWAY_IDLE_TIMEOUT")

	cfg := LoadPoolConfigFromEnv()
	// Should fall back to defaults
	if cfg.MaxIdleConns != 100 {
		t.Errorf("invalid value should use default, got %d", cfg.MaxIdleConns)
	}
	if cfg.MaxConnsPerHost != 10 {
		t.Errorf("invalid value should use default, got %d", cfg.MaxConnsPerHost)
	}
	if cfg.IdleConnTimeout != 90*time.Second {
		t.Errorf("invalid value should use default, got %v", cfg.IdleConnTimeout)
	}
}

func TestNewTransport(t *testing.T) {
	cfg := DefaultPoolConfig()
	tr := NewTransport(cfg)
	if tr.MaxIdleConns != cfg.MaxIdleConns {
		t.Errorf("MaxIdleConns mismatch")
	}
	if tr.MaxIdleConnsPerHost != cfg.MaxConnsPerHost {
		t.Errorf("MaxIdleConnsPerHost mismatch")
	}
	if tr.IdleConnTimeout != cfg.IdleConnTimeout {
		t.Errorf("IdleConnTimeout mismatch")
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true")
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultPoolConfig()
	client := NewClient(cfg, 30*time.Second)
	if client.Timeout != 30*time.Second {
		t.Errorf("timeout = %v", client.Timeout)
	}
	if client.Transport == nil {
		t.Error("transport should not be nil")
	}
	// Verify it's an *http.Transport
	if _, ok := client.Transport.(*http.Transport); !ok {
		t.Error("transport should be *http.Transport")
	}
}
