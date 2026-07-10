// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// TracingConfig configures distributed tracing for the gateway.
type TracingConfig struct {
	ServiceName    string // e.g. "ggid-gateway"
	OTLPEndpoint   string // e.g. "localhost:4317" (empty = disabled)
	TraceIDHeader  string // W3C Traceparent header (default: traceparent)
	SampleRate     float64 // 0.0-1.0 (1.0 = sample all)
	ExportInterval time.Duration
}

// DefaultTracingConfig returns default tracing configuration.
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		ServiceName:    "ggid-gateway",
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		TraceIDHeader:  "traceparent",
		SampleRate:     1.0,
		ExportInterval: 5 * time.Second,
	}
}

// Span represents a distributed tracing span.
type Span struct {
	TraceID     string    `json:"trace_id"`
	SpanID      string    `json:"span_id"`
	ParentID    string    `json:"parent_span_id,omitempty"`
	Name        string    `json:"name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	StatusCode  int       `json:"status_code"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// TraceContext holds tracing state for a request.
type TraceContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Sampled    bool
	Span       *Span
	Exporter   *TraceExporter
}

type traceContextKey struct{}

// TraceExporter collects and exports spans.
type TraceExporter struct {
	spans  chan *Span
	config TracingConfig
	done   chan struct{}
}

// NewTraceExporter creates a background span exporter.
// If OTLPEndpoint is empty, spans are logged to stderr instead.
func NewTraceExporter(cfg TracingConfig) *TraceExporter {
	exp := &TraceExporter{
		spans:  make(chan *Span, 1000),
		config: cfg,
		done:   make(chan struct{}),
	}
	go exp.run()
	return exp
}

func (e *TraceExporter) run() {
	ticker := time.NewTicker(e.config.ExportInterval)
	defer ticker.Stop()
	var batch []*Span

	for {
		select {
		case span := <-e.spans:
			batch = append(batch, span)
			if len(batch) >= 64 {
				e.export(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				e.export(batch)
				batch = nil
			}
		case <-e.done:
			if len(batch) > 0 {
				e.export(batch)
			}
			return
		}
	}
}

// Shutdown stops the exporter and flushes remaining spans.
func (e *TraceExporter) Shutdown() {
	close(e.done)
}

func (e *TraceExporter) export(spans []*Span) {
	if e.config.OTLPEndpoint == "" {
		// Log to stderr for local development
		for _, s := range spans {
			fmt.Fprintf(os.Stderr,
				`{"timestamp":"%s","level":"trace","service":"%s","trace_id":"%s","span_id":"%s","name":"%s","duration_ms":%d,"status":%d}`+"\n",
				s.StartTime.UTC().Format(time.RFC3339Nano),
				e.config.ServiceName,
				s.TraceID, s.SpanID, s.Name,
				s.EndTime.Sub(s.StartTime).Milliseconds(),
				s.StatusCode,
			)
		}
		return
	}
	// In production, this would send OTLP gRPC/HTTP to collector
	// For now, we format as OTLP-compatible JSON
	for _, s := range spans {
		fmt.Fprintf(os.Stderr,
			`{"otel":"trace","trace_id":"%s","span_id":"%s","name":"%s","duration_ms":%d,"endpoint":"%s"}`+"\n",
			s.TraceID, s.SpanID, s.Name,
			s.EndTime.Sub(s.StartTime).Milliseconds(),
			e.config.OTLPEndpoint,
		)
	}
}

// Export sends a span to the exporter.
func (e *TraceExporter) Export(span *Span) {
	if e == nil {
		return
	}
	select {
	case e.spans <- span:
	default: // drop span if buffer full
	}
}

// parseTraceparent parses a W3C traceparent header.
// Format: 00-<trace_id>-<span_id>-<flags>
// Example: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
func parseTraceparent(header string) (traceID, spanID, parentID string, sampled bool) {
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return "", "", "", false
	}
	if parts[0] != "00" {
		return "", "", "", false
	}
	traceID = parts[1]
	spanID = parts[2]
	// parts[3] is flags — "01" means sampled
	sampled = strings.Contains(parts[3], "1")
	return traceID, spanID, "", sampled
}

// formatTraceparent creates a W3C traceparent header value.
func formatTraceparent(traceID, spanID string, sampled bool) string {
	flags := "00"
	if sampled {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", traceID, spanID, flags)
}

// generateTraceID generates a 32-char hex trace ID.
func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateSpanID generates a 16-char hex span ID.
func generateSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// shouldSample determines if a request should be sampled based on SampleRate.
func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}
	// Deterministic sampling: use time-based jitter
	return float64(time.Now().UnixNano()%1000)/1000.0 < rate
}

// TracingMiddleware creates a distributed tracing middleware.
// It parses or generates W3C traceparent headers, creates spans for each request,
// and exports them via the TraceExporter.
func TracingMiddleware(cfg TracingConfig, exporter *TraceExporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse incoming traceparent or generate new trace
			traceID, parentSpanID, _, incomingSampled := parseTraceparent(r.Header.Get(cfg.TraceIDHeader))

			sampled := incomingSampled
			if traceID == "" {
				traceID = generateTraceID()
				sampled = shouldSample(cfg.SampleRate)
			}

			spanID := generateSpanID()

			span := &Span{
				TraceID:   traceID,
				SpanID:    spanID,
				ParentID:  parentSpanID,
				Name:      fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				StartTime: time.Now(),
				Attributes: map[string]string{
					"http.method":     r.Method,
					"http.url":        r.URL.String(),
					"http.host":       r.Host,
					"http.user_agent": r.UserAgent(),
				},
			}

			// Set outgoing traceparent header
			w.Header().Set(cfg.TraceIDHeader, formatTraceparent(traceID, spanID, sampled))

			tc := &TraceContext{
				TraceID:  traceID,
				SpanID:   spanID,
				ParentID: parentSpanID,
				Sampled:  sampled,
				Span:     span,
				Exporter: exporter,
			}
			ctx := context.WithValue(r.Context(), traceContextKey{}, tc)

			// Wrap response writer to capture status
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r.WithContext(ctx))

			span.EndTime = time.Now()
			span.StatusCode = sr.status
			span.Attributes["http.status_code"] = fmt.Sprintf("%d", sr.status)

			if sampled {
				exporter.Export(span)
			}
		})
	}
}

// TraceFromRequest extracts the trace context from a request.
func TraceFromRequest(r *http.Request) (*TraceContext, bool) {
	tc, ok := r.Context().Value(traceContextKey{}).(*TraceContext)
	return tc, ok
}

// ChildSpan creates a child span within an existing trace context.
func (tc *TraceContext) ChildSpan(name string) *Span {
	if tc == nil {
		return nil
	}
	return &Span{
		TraceID:   tc.TraceID,
		SpanID:    generateSpanID(),
		ParentID:  tc.SpanID,
		Name:      name,
		StartTime: time.Now(),
	}
}

// FinishSpan completes a span and exports it.
func (tc *TraceContext) FinishSpan(span *Span, statusCode int) {
	if tc == nil || span == nil {
		return
	}
	span.EndTime = time.Now()
	span.StatusCode = statusCode
	if tc.Sampled && tc.Exporter != nil {
		tc.Exporter.Export(span)
	}
}
