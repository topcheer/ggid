package transport

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// PoolConfig holds tunable parameters for the HTTP connection pool.
type PoolConfig struct {
	MaxIdleConns        int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	TLSHandshakeTimeout time.Duration
	DialTimeout         time.Duration
	KeepAlive           time.Duration
}

// DefaultPoolConfig returns production-ready defaults.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConns:        100,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DialTimeout:         5 * time.Second,
		KeepAlive:           30 * time.Second,
	}
}

// LoadPoolConfigFromEnv reads connection pool settings from environment variables:
//   GATEWAY_MAX_IDLE_CONNS (default 100)
//   GATEWAY_MAX_CONNS_PER_HOST (default 10)
//   GATEWAY_IDLE_TIMEOUT (seconds, default 90)
func LoadPoolConfigFromEnv() PoolConfig {
	cfg := DefaultPoolConfig()

	if v := os.Getenv("GATEWAY_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxIdleConns = n
		}
	}
	if v := os.Getenv("GATEWAY_MAX_CONNS_PER_HOST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxConnsPerHost = n
		}
	}
	if v := os.Getenv("GATEWAY_IDLE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.IdleConnTimeout = time.Duration(n) * time.Second
		}
	}

	return cfg
}

// NewTransport creates an *http.Transport configured with the given pool settings.
func NewTransport(cfg PoolConfig) *http.Transport {
	return &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:   true,
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
	}
}

// NewClient creates an *http.Client with the configured transport and the given timeout.
func NewClient(cfg PoolConfig, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: NewTransport(cfg),
		Timeout:   timeout,
	}
}
