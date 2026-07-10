package middleware

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"time"
)

// ShadowTrafficConfig configures dark launch / shadow traffic mirroring.
type ShadowTrafficConfig struct {
	// ShadowBackend is the URL of the new version backend to mirror traffic to.
	ShadowBackend string
	// Percentage of requests to mirror (0-100).
	Percentage int
	// Methods to mirror (empty = all methods).
	Methods []string
	// Timeout for shadow requests.
	Timeout time.Duration
	// DropPercentage of shadow responses (always dropped — results are never returned).
	// This field exists for documentation; shadow responses are always discarded.
}

// ShadowTrafficMirrors copies a percentage of requests to a shadow backend
// for dark launch testing. Responses from the shadow backend are discarded.
// The original request proceeds normally without any delay.
type ShadowTrafficMirror struct {
	config ShadowTrafficConfig
	client *http.Client
	mu     sync.RWMutex
	stats  ShadowStats
}

// ShadowStats tracks shadow traffic statistics.
type ShadowStats struct {
	TotalMirrored  int64
	TotalErrors    int64
	TotalLatencyMs int64
}

// NewShadowTrafficMirror creates a new shadow traffic mirror.
func NewShadowTrafficMirror(cfg ShadowTrafficConfig) *ShadowTrafficMirror {
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.Percentage < 0 {
		cfg.Percentage = 0
	}
	if cfg.Percentage > 100 {
		cfg.Percentage = 100
	}
	return &ShadowTrafficMirror{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// ShadowMiddleware mirrors a percentage of requests to a shadow backend.
// The original request is NOT affected — shadow requests run asynchronously
// and their responses are discarded.
func ShadowMiddleware(mirror *ShadowTrafficMirror) func(http.Handler) http.Handler {
	if mirror == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mirror the request asynchronously
			if mirror.shouldMirror(r) {
				go mirror.sendShadow(r)
			}
			// Process the original request normally
			next.ServeHTTP(w, r)
		})
	}
}

func (m *ShadowTrafficMirror) shouldMirror(r *http.Request) bool {
	// Check for per-request shadow backend via header
	if backend := r.Header.Get("X-Shadow-Backend"); backend != "" {
		return true
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check method filter
	if len(m.config.Methods) > 0 {
		found := false
		for _, method := range m.config.Methods {
			if r.Method == method {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check percentage
	if m.config.Percentage <= 0 {
		return false
	}
	if m.config.Percentage >= 100 {
		return true
	}

	// Deterministic percentage based on time (no crypto needed)
	return int(time.Now().UnixNano()%100) < m.config.Percentage
}

func (m *ShadowTrafficMirror) sendShadow(r *http.Request) {
	m.mu.Lock()
	m.stats.TotalMirrored++
	m.mu.Unlock()

	start := time.Now()

	// Determine target: per-request header overrides config
	shadowBackend := r.Header.Get("X-Shadow-Backend")
	if shadowBackend == "" {
		m.mu.RLock()
		shadowBackend = m.config.ShadowBackend
		m.mu.RUnlock()
	}
	if shadowBackend == "" {
		return
	}

	shadowURL := shadowBackend + r.URL.Path
	if r.URL.RawQuery != "" {
		shadowURL += "?" + r.URL.RawQuery
	}

	// Read body for non-GET requests
	var bodyBytes []byte
	if r.Body != nil && r.Method != http.MethodGet {
		bodyBytes, _ = readAll(r.Body)
		r.Body.Close()
	}

	shadowReq, err := http.NewRequestWithContext(r.Context(), r.Method, shadowURL, bytes.NewReader(bodyBytes))
	if err != nil {
		m.recordError()
		return
	}

	// Copy headers
	for k, v := range r.Header {
		shadowReq.Header[k] = v
	}
	shadowReq.Header.Set("X-Shadow-Traffic", "true")

	resp, err := m.client.Do(shadowReq)
	if err != nil {
		m.recordError()
		return
	}
	resp.Body.Close()

	m.mu.Lock()
	m.stats.TotalLatencyMs += time.Since(start).Milliseconds()
	m.mu.Unlock()
}

func (m *ShadowTrafficMirror) recordError() {
	m.mu.Lock()
	m.stats.TotalErrors++
	m.mu.Unlock()
}

// GetStats returns the current shadow traffic statistics.
func (m *ShadowTrafficMirror) GetStats() ShadowStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// SetPercentage updates the mirror percentage at runtime.
func (m *ShadowTrafficMirror) SetPercentage(p int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	m.config.Percentage = p
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

// Ensure log is used
var _ = log.Printf
