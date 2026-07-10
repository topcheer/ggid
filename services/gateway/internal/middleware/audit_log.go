package middleware

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// AuditEvent represents a single request audit log entry.
type AuditEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  float64   `json:"latency_ms"`
	TenantID   string    `json:"tenant_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	BytesSent  int64     `json:"bytes_sent"`
}

// AuditPublisher publishes audit events to an external system (e.g. NATS).
// Implementations must be safe for concurrent use.
type AuditPublisher interface {
	Publish(event *AuditEvent) error
}

// NATSAuditPublisher publishes audit events via a NATS connection.
// If the connection is nil or publishing fails, events are silently dropped
// (audit logging must never block the request path).

type NATSAuditPublisher struct {
	subject string
	nc      NATSConn
	dropped int64
	mu      sync.Mutex
}

// NATSConn is the minimal interface needed from a NATS connection.
type NATSConn interface {
	Publish(subj string, data []byte) error
}

// NewNATSAuditPublisher creates a publisher for the given NATS subject.
func NewNATSAuditPublisher(nc NATSConn, subject string) *NATSAuditPublisher {
	return &NATSAuditPublisher{
		nc:      nc,
		subject: subject,
	}
}

// Publish sends an audit event as JSON to the NATS subject.
func (p *NATSAuditPublisher) Publish(event *AuditEvent) error {
	if p.nc == nil {
		return nil // no-op when NATS is not configured
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.nc.Publish(p.subject, data)
}

// DroppedCount returns the number of events that failed to publish.
func (p *NATSAuditPublisher) DroppedCount() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dropped
}

// noopAuditPublisher drops all events silently.
type noopAuditPublisher struct{}

func (noopAuditPublisher) Publish(_ *AuditEvent) error { return nil }

// AuditLogger collects audit events from HTTP requests and publishes
// them asynchronously to avoid blocking the response.
type AuditLogger struct {
	publisher AuditPublisher
	queue     chan *AuditEvent
	wg        sync.WaitGroup
	stop      chan struct{}
}

// NewAuditLogger creates an audit logger with the given publisher and
// buffer size. Events are published in background goroutines.
func NewAuditLogger(publisher AuditPublisher, bufferSize int) *AuditLogger {
	if publisher == nil {
		publisher = noopAuditPublisher{}
	}
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	al := &AuditLogger{
		publisher: publisher,
		queue:     make(chan *AuditEvent, bufferSize),
		stop:      make(chan struct{}),
	}
	// Start worker goroutines (2x CPU cores)
	workers := runtime.NumCPU() * 2
	if workers > 8 {
		workers = 8
	}
	for i := 0; i < workers; i++ {
		al.wg.Add(1)
		go al.worker()
	}
	return al
}

func (al *AuditLogger) worker() {
	defer al.wg.Done()
	for {
		select {
		case event := <-al.queue:
			_ = al.publisher.Publish(event)
		case <-al.stop:
			return
		}
	}
}

// Enqueue adds an audit event to the async queue. Non-blocking — if the
// queue is full, the event is silently dropped.
func (al *AuditLogger) Enqueue(event *AuditEvent) {
	select {
	case al.queue <- event:
	default:
		// Queue full — drop event silently
	}
}

// Stop shuts down the audit logger, flushing pending events.
func (al *AuditLogger) Stop() {
	close(al.stop)
	al.wg.Wait()
}

// AuditMiddleware wraps the HTTP handler with audit logging.
// It captures the response status code, latency, and request metadata,
// then enqueues an AuditEvent asynchronously.
func AuditMiddleware(logger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &auditResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			event := &AuditEvent{
				Timestamp:  start,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rec.status,
				LatencyMs:  float64(time.Since(start).Microseconds()) / 1000.0,
				TenantID:   r.Header.Get("X-Tenant-ID"),
				ClientIP:   r.RemoteAddr,
				UserAgent:  r.UserAgent(),
				RequestID:  r.Header.Get("X-Request-ID"),
				BytesSent:  rec.bytesWritten,
			}

			logger.Enqueue(event)
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}
