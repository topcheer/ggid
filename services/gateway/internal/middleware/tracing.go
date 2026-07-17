package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SpanContext holds W3C trace context.
type SpanContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Operation  string
	StartTime  time.Time
	Attributes map[string]string
}

// traceKey is the context key for the current span.
type traceKey struct{}

// GenerateTraceID creates a W3C-compliant 32-char hex trace ID.
func GenerateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateSpanID creates a 16-char hex span ID.
func GenerateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// TraceMiddleware adds W3C trace context to every request.
// Sampling: 10% of requests are traced (configurable), 100% of errors.
var (
	sampleRate    = 0.1 // 10% sampling
	sampleRateMu  sync.RWMutex
)

// SetSampleRate configures the trace sampling rate (0.0-1.0).
func SetSampleRate(rate float64) {
	sampleRateMu.Lock()
	if rate < 0 { rate = 0 }
	if rate > 1 { rate = 1 }
	sampleRate = rate
	sampleRateMu.Unlock()
}

func getSampleRate() float64 {
	sampleRateMu.RLock()
	defer sampleRateMu.RUnlock()
	return sampleRate
}

// TraceMiddleware instruments HTTP requests with distributed tracing.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health checks.
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract or generate trace ID (W3C traceparent header).
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			// Check W3C traceparent.
			if tp := r.Header.Get("Traceparent"); len(tp) >= 35 {
				traceID = tp[3:35]
			}
		}
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		spanID := GenerateSpanID()

		// Sampling decision.
		sampled := false
		if r.Header.Get("X-Trace-Forced") == "true" {
			sampled = true
		} else {
			// Deterministic sampling based on trace ID hash.
			hash := 0
			for _, c := range traceID {
				hash = (hash*31 + int(c)) % 1000
			}
			sampled = float64(hash)/1000 < getSampleRate()
		}

		span := &SpanContext{
			TraceID:   traceID,
			SpanID:    spanID,
			Operation: fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			StartTime: time.Now(),
			Attributes: map[string]string{
				"http.method": r.Method,
				"http.url":    r.URL.Path,
				"sampled":     fmt.Sprintf("%v", sampled),
			},
		}

		// Set trace headers on response.
		w.Header().Set("X-Trace-Id", traceID)
		w.Header().Set("X-Span-Id", spanID)

		// Inject span into context.
		ctx := context.WithValue(r.Context(), traceKey{}, span)

		// Wrap response writer to capture status code.
		sw := &statusWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(sw, r.WithContext(ctx))

		// Record span completion.
		duration := time.Since(span.StartTime)
		span.Attributes["http.status_code"] = fmt.Sprintf("%d", sw.status)
		span.Attributes["duration_ms"] = fmt.Sprintf("%d", duration.Milliseconds())

		// Force trace on errors.
		if sw.status >= 500 {
			sampled = true
		}

		if sampled {
			RecordSpan(span, sw.status, duration)
		}
	})
}

// SpanFromContext extracts the current span from context.
func SpanFromContext(ctx context.Context) *SpanContext {
	if v, ok := ctx.Value(traceKey{}).(*SpanContext); ok {
		return v
	}
	return nil
}

// SetSpanAttribute adds an attribute to the current span.
func SetSpanAttribute(ctx context.Context, key, value string) {
	if span := SpanFromContext(ctx); span != nil {
		span.Attributes[key] = value
	}
}

// --- Trace Store (in-memory ring buffer) ---

type TraceRecord struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Operation  string            `json:"operation"`
	StatusCode int               `json:"status_code"`
	Duration   time.Duration     `json:"duration"`
	Attributes map[string]string `json:"attributes"`
	Timestamp  time.Time         `json:"timestamp"`
}

var (
	traceStore   []TraceRecord
	traceStoreMu sync.Mutex
	maxTraces    = 10000
)

// RecordSpan stores a completed span.
func RecordSpan(span *SpanContext, statusCode int, duration time.Duration) {
	traceStoreMu.Lock()
	defer traceStoreMu.Unlock()
	rec := TraceRecord{
		TraceID: span.TraceID, SpanID: span.SpanID,
		Operation: span.Operation, StatusCode: statusCode,
		Duration: duration, Attributes: span.Attributes,
		Timestamp: time.Now(),
	}
	traceStore = append(traceStore, rec)
	if len(traceStore) > maxTraces {
		traceStore = traceStore[len(traceStore)-maxTraces:]
	}
}

// GetRecentTraces returns the most recent trace records.
func GetRecentTraces(limit int) []TraceRecord {
	traceStoreMu.Lock()
	defer traceStoreMu.Unlock()
	if limit <= 0 || limit > len(traceStore) {
		limit = len(traceStore)
	}
	if limit == 0 {
		return []TraceRecord{}
	}
	start := len(traceStore) - limit
	result := make([]TraceRecord, limit)
	copy(result, traceStore[start:])
	return result
}

// statusWriter wraps http.ResponseWriter to capture status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}
