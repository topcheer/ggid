package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// LogEntry is a structured log entry for an HTTP request.
type LogEntry struct {
	Timestamp  string `json:"timestamp"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Latency   string `json:"latency"`
	TenantID  string `json:"tenant_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	IP        string `json:"ip,omitempty"`
	BytesOut  int    `json:"bytes_out"`
}

// LogLevel determines which log function to use.
type LogLevel int

const (
	LogLevelInfo  LogLevel = iota
	LogLevelWarn
	LogLevelError
)

// Logger is the interface for structured logging.
type Logger interface {
	Info(entry LogEntry)
	Warn(entry LogEntry)
	Error(entry LogEntry)
}

// JSONLogger writes structured JSON log entries to an http.ResponseWriter-like sink.
// In production, this would write to stdout or a log aggregation system.
type JSONLogger struct {
	writer func(string)
}

// NewJSONLogger creates a logger that calls the given function for each log line.
func NewJSONLogger(writer func(string)) *JSONLogger {
	return &JSONLogger{writer: writer}
}

func (l *JSONLogger) write(level string, entry LogEntry) {
	if l.writer == nil {
		return
	}
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	l.writer(`{"level":"` + level + `","` + strconv.Itoa(len(data)) + `":` + string(data) + `}`)
}

func (l *JSONLogger) Info(entry LogEntry)  { l.write("info", entry) }
func (l *JSONLogger) Warn(entry LogEntry)  { l.write("warn", entry) }
func (l *JSONLogger) Error(entry LogEntry) { l.write("error", entry) }

// statusLogLevel determines the log level based on HTTP status code.
func statusLogLevel(status int) LogLevel {
	if status >= 500 {
		return LogLevelError
	}
	if status >= 400 {
		return LogLevelWarn
	}
	return LogLevelInfo
}

// loggingResponseWriter captures status code and bytes written.
type loggingResponseWriter struct {
	http.ResponseWriter
	status   int
	bytes    int
	captured bool
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	if w.captured {
		return
	}
	w.status = code
	w.captured = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if !w.captured {
		w.status = http.StatusOK
		w.captured = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// RequestLogging returns middleware that logs each request as structured JSON.
// It uses info level for 2xx, warn for 4xx, error for 5xx.
// Health check requests (/healthz) are not logged.
func RequestLogging(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(lw, r)

			entry := LogEntry{
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    lw.status,
				Latency:   time.Since(start).String(),
				BytesOut:  lw.bytes,
			}

			// Extract context values
			if tid, ok := TenantIDFromRequest(r); ok {
				entry.TenantID = tid
			}
			if rid, ok := r.Context().Value(RequestIDKey).(string); ok {
				entry.RequestID = rid
			}
			entry.IP = ClientIP(r)

			switch statusLogLevel(lw.status) {
			case LogLevelError:
				logger.Error(entry)
			case LogLevelWarn:
				logger.Warn(entry)
			default:
				logger.Info(entry)
			}
		})
	}
}

// NoopLogger discards all log entries. Useful for testing.
type NoopLogger struct{}

func (NoopLogger) Info(LogEntry)  {}
func (NoopLogger) Warn(LogEntry)  {}
func (NoopLogger) Error(LogEntry) {}

// CapturingLogger captures log entries for testing.
type CapturingLogger struct {
	Entries []LogEntry
	Levels  []string
}

func (c *CapturingLogger) Info(entry LogEntry) {
	c.Entries = append(c.Entries, entry)
	c.Levels = append(c.Levels, "info")
}
func (c *CapturingLogger) Warn(entry LogEntry) {
	c.Entries = append(c.Entries, entry)
	c.Levels = append(c.Levels, "warn")
}
func (c *CapturingLogger) Error(entry LogEntry) {
	c.Entries = append(c.Entries, entry)
	c.Levels = append(c.Levels, "error")
}
