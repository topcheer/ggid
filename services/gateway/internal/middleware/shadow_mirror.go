package middleware

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ShadowMirrorConfig configures traffic mirroring to a canary backend.
type ShadowMirrorConfig struct {
	// TargetURL is the canary/shadow backend URL.
	TargetURL string
	// Percentage of traffic to mirror (0-100).
	Percentage int
	// Timeout for shadow requests (default 5s).
	Timeout time.Duration
	// OnResult is called with the latency difference between primary and shadow.
	OnResult func(primaryLatency, shadowLatency time.Duration, status int)
}

// ShadowMirror duplicates a percentage of traffic to a canary backend asynchronously.
// Shadow requests never block or affect the primary response.
type ShadowMirror struct {
	cfg     *ShadowMirrorConfig
	client  *http.Client
	total   atomic.Uint64
	mirrored atomic.Uint64
	mu      sync.Mutex
}

// NewShadowMirror creates a shadow traffic mirror.
func NewShadowMirror(cfg *ShadowMirrorConfig) *ShadowMirror {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.Percentage < 0 {
		cfg.Percentage = 0
	}
	if cfg.Percentage > 100 {
		cfg.Percentage = 100
	}

	return &ShadowMirror{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Middleware returns HTTP middleware that mirrors traffic.
func (sm *ShadowMirror) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Determine if this request should be mirrored
		shouldMirror := sm.cfg.TargetURL != "" && rand.Intn(100) < sm.cfg.Percentage
		sm.total.Add(1)

		if shouldMirror {
			// Clone the request body for shadow
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			// Fire shadow request asynchronously
			go sm.sendShadow(r.Clone(r.Context()), bodyBytes, start)
		}

		// Serve primary request (never blocked by shadow)
		next.ServeHTTP(w, r)
	})
}

func (sm *ShadowMirror) sendShadow(req *http.Request, body []byte, primaryStart time.Time) {
	defer func() { recover() }() // never panic

	sm.mirrored.Add(1)

	// Build shadow request
	shadowReq, err := http.NewRequestWithContext(req.Context(), req.Method, sm.cfg.TargetURL+req.URL.RequestURI(), bytes.NewReader(body))
	if err != nil {
		return
	}

	// Copy headers
	for k, v := range req.Header {
		shadowReq.Header[k] = v
	}
	shadowReq.Header.Set("X-Shadow-Traffic", "true")

	shadowStart := time.Now()
	resp, err := sm.client.Do(shadowReq)
	shadowLatency := time.Since(shadowStart)
	primaryLatency := time.Since(primaryStart)

	if err != nil {
		if sm.cfg.OnResult != nil {
			sm.cfg.OnResult(primaryLatency, shadowLatency, 0)
		}
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if sm.cfg.OnResult != nil {
		sm.cfg.OnResult(primaryLatency, shadowLatency, resp.StatusCode)
	}
}

// Stats returns total requests seen and mirrored count.
func (sm *ShadowMirror) Stats() (total, mirrored uint64) {
	return sm.total.Load(), sm.mirrored.Load()
}
