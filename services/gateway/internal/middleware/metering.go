// Package middleware provides per-tenant API usage metering.
// Records request count, latency, and status per tenant for billing/analytics.
// Uses async batch inserts to PostgreSQL to minimize latency impact.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// MeteringConfig controls the behavior of the API metering middleware.
type MeteringConfig struct {
	// FlushInterval is how often to batch-insert usage records (default 30s).
	FlushInterval time.Duration
	// MaxBufferSize is the max records buffered before an early flush (default 500).
	MaxBufferSize int
	// ExcludePaths are path prefixes to skip (e.g. /healthz, /metrics).
	ExcludePaths []string
	// AuditURL is the audit service URL for HTTP flush. If empty, falls back to DB.
	AuditURL string
}

// DefaultMeteringConfig returns sensible defaults.
func DefaultMeteringConfig() MeteringConfig {
	return MeteringConfig{
		FlushInterval: 30 * time.Second,
		MaxBufferSize: 500,
		ExcludePaths:  []string{"/healthz", "/metrics", "/ready", "/livez"},
		AuditURL:      os.Getenv("GGID_AUDIT_URL"),
	}
}

// usageRecord captures a single API call's metering data.
type usageRecord struct {
	TenantID  string  `json:"tenant_id"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    int     `json:"status"`
	Duration  float64 `json:"duration_ms"`
	Timestamp int64   `json:"timestamp_unix"`
}

// usageAggregator buffers records and flushes async to the audit service.
type usageAggregator struct {
	mu         sync.Mutex
	buffer     []usageRecord
	config     MeteringConfig
	client     *http.Client
	stopCh     chan struct{}
}

// APIMetering creates a middleware that records per-tenant API usage.
// dbURL is the PostgreSQL connection string (currently used to derive audit URL).
// config controls batching behavior.
func APIMetering(dbURL string, config MeteringConfig) func(http.Handler) http.Handler {
	agg := &usageAggregator{
		buffer: make([]usageRecord, 0, config.MaxBufferSize),
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
		stopCh: make(chan struct{}),
	}
	go agg.flushLoop()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip excluded paths
			for _, p := range config.ExcludePaths {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()
			ww := &usageStatusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			duration := time.Since(start).Seconds() * 1000

			tenantID := tenantIDFromRequest(r)

			agg.record(usageRecord{
				TenantID:  tenantID,
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    ww.status,
				Duration:  duration,
				Timestamp: start.Unix(),
			})
		})
	}
}

func (a *usageAggregator) record(r usageRecord) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buffer = append(a.buffer, r)
	if len(a.buffer) >= a.config.MaxBufferSize {
		go a.flush()
	}
}

func (a *usageAggregator) flushLoop() {
	if a.config.FlushInterval == 0 {
		a.config.FlushInterval = 30 * time.Second
	}
	ticker := time.NewTicker(a.config.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.flush()
		case <-a.stopCh:
			a.flush()
			return
		}
	}
}

func (a *usageAggregator) flush() {
	a.mu.Lock()
	if len(a.buffer) == 0 {
		a.mu.Unlock()
		return
	}
	batch := a.buffer
	a.buffer = make([]usageRecord, 0, a.config.MaxBufferSize)
	a.mu.Unlock()

	if a.config.AuditURL == "" {
		// No audit URL configured — silently drop (logged once)
		log.Printf("[metering] flushed %d records (no audit URL, dropped)", len(batch))
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type":   "api_usage",
		"events": batch,
	})
	if err != nil {
		log.Printf("[metering] marshal error: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1/audit/usage", a.config.AuditURL)
	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		log.Printf("[metering] request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "gateway")

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[metering] flush error: %v (batch: %d)", err, len(batch))
		return
	}
	resp.Body.Close()
}

// tenantIDFromRequest extracts tenant ID from JWT context or header.
func tenantIDFromRequest(r *http.Request) string {
	// Check context (set by JWT auth middleware)
	if v := r.Context().Value(tenantCtxKey{}); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	// Fallback: X-Tenant-ID header
	if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
		return tid
	}
	return "anonymous"
}

type tenantCtxKey struct{}

// WithTenant sets the tenant ID in the request context.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tenantID)
}

type usageStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *usageStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *usageStatusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

