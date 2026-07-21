package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// TimeoutConfig defines per-route timeout settings.
type TimeoutConfig struct {
	Default     time.Duration         // default timeout for all routes (e.g. 30s)
	RouteConfigs map[string]time.Duration // per-route overrides keyed by path prefix
}

// DefaultTimeoutConfig returns a config with 30s default timeout.
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Default: 30 * time.Second,
		RouteConfigs: map[string]time.Duration{
			"/api/v1/auth/verify":    10 * time.Second,
			"/api/v1/auth/register": 15 * time.Second,
			"/api/v1/audit":         60 * time.Second,
		},
	}
}

// timeoutResponseWriter captures whether headers were already written.
type timeoutResponseWriter struct {
	http.ResponseWriter
	mu        sync.Mutex
	timedOut  bool
	headerSet bool
}

func (w *timeoutResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}
	w.headerSet = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *timeoutResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, nil
	}
	return w.ResponseWriter.Write(b)
}

// TimeoutMiddleware returns middleware that enforces per-route request timeouts.
// On timeout, it returns 504 Gateway Timeout with a JSON error body and cancels
// the upstream context.
func TimeoutMiddleware(cfg *TimeoutConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultTimeoutConfig()
	}
	if cfg.Default <= 0 {
		cfg.Default = 30 * time.Second
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine timeout for this route
			timeout := cfg.Default
			for prefix, dur := range cfg.RouteConfigs {
				if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
					timeout = dur
					break
			}
			}

			// Skip timeout for health checks and WebSocket upgrades
			if r.URL.Path == "/healthz" || isWebSocketUpgrade(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			tw := &timeoutResponseWriter{ResponseWriter: w}

			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				tw.mu.Lock()
				tw.timedOut = true
				tw.mu.Unlock()

				if !tw.headerSet {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Request-Timeout", timeout.String())
					w.WriteHeader(http.StatusGatewayTimeout)
					w.Write([]byte(`{"error":"gateway_timeout","message":"request exceeded timeout","timeout":"` + timeout.String() + `"}`))
				}
			}
		})
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade.
func isWebSocketUpgrade(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket" ||
		(r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") != "")
}

// GetTimeoutForRoute returns the timeout duration for a given path.
func (cfg *TimeoutConfig) GetTimeoutForRoute(path string) time.Duration {
	for prefix, dur := range cfg.RouteConfigs {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return dur
		}
	}
	return cfg.Default
}
