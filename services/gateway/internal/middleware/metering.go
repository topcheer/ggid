// Package middleware provides per-tenant API usage metering.
// Records request count, latency, and status per tenant for billing/analytics.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UsageRecord captures a single API call's metering data.
type UsageRecord struct {
	TenantID  string    `json:"tenant_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	Duration  float64   `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

// UsageAggregator collects usage records in memory and flushes
// them in batches to the audit service via HTTP.
type UsageAggregator struct {
	mu         sync.Mutex
	buffer     []UsageRecord
	maxBuffer  int
	flushEvery time.Duration
	auditURL   string
	client     *http.Client
	stopCh     chan struct{}
}

// NewUsageAggregator creates a background usage aggregator.
// auditURL should point to the audit service's ingestion endpoint.
func NewUsageAggregator(auditURL string) *UsageAggregator {
	ua := &UsageAggregator{
		buffer:     make([]UsageRecord, 0, 500),
		maxBuffer:  500,
		flushEvery: 30 * time.Second,
		auditURL:   auditURL,
		client:     &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
	}
	go ua.flushLoop()
	return ua
}

// Record adds a usage entry to the buffer.
func (ua *UsageAggregator) Record(r UsageRecord) {
	ua.mu.Lock()
	defer ua.mu.Unlock()
	ua.buffer = append(ua.buffer, r)
	if len(ua.buffer) >= ua.maxBuffer {
		go ua.flush()
	}
}

func (ua *UsageAggregator) flushLoop() {
	ticker := time.NewTicker(ua.flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ua.flush()
		case <-ua.stopCh:
			ua.flush()
			return
		}
	}
}

func (ua *UsageAggregator) flush() {
	ua.mu.Lock()
	if len(ua.buffer) == 0 {
		ua.mu.Unlock()
		return
	}
	batch := ua.buffer
	ua.buffer = make([]UsageRecord, 0, ua.maxBuffer)
	ua.mu.Unlock()

	// POST batch to audit service
	payload, err := json.Marshal(map[string]interface{}{
		"events": batch,
		"type":   "api_usage",
	})
	if err != nil {
		log.Printf("[usage] marshal error: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/v1/audit/usage", ua.auditURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Printf("[usage] request error: %v", err)
		return
	}
	req.Body = http.NoBody
	req.Body = stringReader(string(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "gateway")

	resp, err := ua.client.Do(req)
	if err != nil {
		log.Printf("[usage] flush error: %v (batch size: %d)", err, len(batch))
		return
	}
	resp.Body.Close()
}

func (ua *UsageAggregator) Stop() {
	close(ua.stopCh)
}

// MeteringMiddleware records per-tenant API usage.
// tenantID is extracted from the request context (set by JWT auth middleware).
// The aggregator handles async batching and flushing to the audit service.
func MeteringMiddleware(aggregator *UsageAggregator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health/metrics endpoints
		path := r.URL.Path
		if path == "/healthz" || path == "/metrics" || path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ww := &usageWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		duration := time.Since(start).Seconds() * 1000 // ms

		// Extract tenant ID from context
		tenantID := tenantFromContext(r.Context())
		if tenantID == "" {
			tenantID = "anonymous"
		}

		aggregator.Record(UsageRecord{
			TenantID:  tenantID,
			Method:    r.Method,
			Path:      path,
			Status:    ww.status,
			Duration:  duration,
			Timestamp: start,
		})
	})
}

type usageWriter struct {
	http.ResponseWriter
	status int
}

func (w *usageWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// tenantFromContext extracts the tenant ID from the request context.
// The JWT middleware sets this via context.WithValue.
type tenantCtxKey struct{}

func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tenantID)
}

func tenantFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tenantCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// Ensure we always have a unique request ID for correlation
func requestID() string {
	return uuid.New().String()
}

// stringReader wraps a string as an io.ReadCloser for request body
type stringReader struct {
	s   string
	pos int
}

func (sr *stringReader) Read(p []byte) (int, error) {
	if sr.pos >= len(sr.s) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, sr.s[sr.pos:])
	sr.pos += n
	return n, nil
}
func (sr *stringReader) Close() error { return nil }

func stringReader2(s string) *stringReader { return &stringReader{s: s} }
