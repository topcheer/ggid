// Package tracing provides OpenTelemetry-style request tracing middleware.
// It uses a lightweight span model that propagates trace context via
// standard W3C Trace Context headers (traceparent/tracestate).
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

// SpanContextKey is the context key for the current span.
var SpanContextKey spanCtx = "span"

type spanCtx string

// Span represents a single request trace span.
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	Duration   time.Duration
	Attributes map[string]string
}

// TraceContext holds trace IDs for propagation.
type TraceContext struct {
	TraceID  string
	SpanID   string
	ParentID string
}

// FromContext extracts the trace context from a request context.
func FromContext(ctx context.Context) (*TraceContext, bool) {
	tc, ok := ctx.Value(SpanContextKey).(*TraceContext)
	return tc, ok
}

// Tracing creates a span for each request, propagating trace context.
// If the incoming request has a traceparent header, it continues that trace.
// Otherwise it starts a new root span.
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc := extractOrCreateTrace(r.Header.Get("traceparent"))

		// Store in context for downstream handlers
		ctx := context.WithValue(r.Context(), SpanContextKey, tc)
		r = r.WithContext(ctx)

		// Record span
		start := time.Now()
		rw := &tracingWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		// Set trace headers on response for client-side correlation
		w.Header().Set("X-Trace-ID", tc.TraceID)
		w.Header().Set("X-Span-ID", tc.SpanID)

		// Span is complete — in production this would be exported to Jaeger/Zipkin
		span := &Span{
			TraceID:    tc.TraceID,
			SpanID:     tc.SpanID,
			ParentID:   tc.ParentID,
			Name:       r.Method + " " + r.URL.Path,
			StartTime:  start,
			Duration:   duration,
			Attributes: map[string]string{
				"http.method":     r.Method,
				"http.url":        r.URL.String(),
				"http.status_code": statusCodeStr(rw.statusCode),
			},
		}
		_ = span // export to collector in production

		// Propagate to backend via traceparent header
		r.Header.Set("traceparent", formatTraceparent(tc))
	})
}

// extractOrCreateTrace parses a W3C traceparent header or creates a new trace.
// Format: 00-<traceId>-<spanId>-<flags>
func extractOrCreateTrace(traceparent string) *TraceContext {
	if traceparent != "" && len(traceparent) >= 55 {
		// W3C format: 00-{traceId32}-{spanId16}-{flags2}
		parts := splitDash(traceparent)
		if len(parts) >= 4 && parts[0] == "00" {
			return &TraceContext{
				TraceID:  parts[1],
				SpanID:   generateSpanID(),
				ParentID: parts[2],
			}
		}
	}
	return &TraceContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
	}
}

func formatTraceparent(tc *TraceContext) string {
	return "00-" + tc.TraceID + "-" + tc.SpanID + "-01"
}

func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func splitDash(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == '-' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func statusCodeStr(code int) string {
	return strconv.Itoa(code)
}

type tracingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (tw *tracingWriter) WriteHeader(code int) {
	tw.statusCode = code
	tw.ResponseWriter.WriteHeader(code)
}
