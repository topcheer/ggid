// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StructuredLogger is a JSON structured logger for HTTP requests.
// It outputs one JSON object per log line, suitable for ingestion by
// log aggregation systems (ELK, Loki, Datadog, etc.).
type StructuredLogger struct {
	logger  *log.Logger
	service string
}

// NewStructuredLogger creates a structured logger writing to stderr.
func NewStructuredLogger(service string) *StructuredLogger {
	return &StructuredLogger{
		logger:  log.New(os.Stderr, "", 0), // no prefix, no flags — raw JSON
		service: service,
	}
}

// LogRecord is a single structured log entry.
type LogRecord struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Service    string `json:"service"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Size       int    `json:"size"`
	Duration   string `json:"duration"`
	RequestID  string `json:"request_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	Message    string `json:"message,omitempty"`
}

// PanicRecord is a structured panic recovery log entry.
type PanicRecord struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Message   string `json:"message"`
	Panic     string `json:"panic"`
	Stack     string `json:"stack"`
	RequestID string `json:"request_id,omitempty"`
	Method    string `json:"method"`
	Path      string `json:"path"`
}

// Emit writes a log record as JSON to the underlying logger.
func (sl *StructuredLogger) Emit(rec any) {
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	sl.logger.Println(string(data))
}

// RequestLogger is structured logging middleware that replaces the basic
// log.Printf-based Logging middleware.  It outputs one JSON line per request.
func RequestLogger(sl *StructuredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)

			requestID, _ := r.Context().Value(RequestIDKey).(string)
			tenantID, _ := TenantIDFromRequest(r)
			userID, _ := r.Context().Value(UserIDKey).(string)

			rec := LogRecord{
				Timestamp:  start.UTC().Format(time.RFC3339Nano),
				Level:      levelFromStatus(sr.status),
				Service:    sl.service,
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     sr.status,
				Size:       sr.size,
				Duration:   time.Since(start).String(),
				RequestID:  requestID,
				TenantID:   tenantID,
				UserID:     userID,
				RemoteAddr: r.RemoteAddr,
			}
			sl.Emit(rec)
		})
	}
}

// levelFromStatus maps HTTP status codes to log levels.
func levelFromStatus(status int) string {
	switch {
	case status >= 500:
		return "error"
	case status >= 400:
		return "warn"
	default:
		return "info"
	}
}

// PanicRecovery is middleware that catches panics from downstream handlers,
// logs a structured panic record, and returns a clean 500 response.
// Without this, a panic would crash the entire gateway process.
func PanicRecovery(sl *StructuredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
			if rv := recover(); rv != nil {
					requestID, _ := r.Context().Value(RequestIDKey).(string)
					svcName := ""
					if sl != nil {
						svcName = sl.service
					}
					panicRec := PanicRecord{
						Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
						Level:     "error",
						Service:   svcName,
						Message:   "panic recovered",
						Panic:     fmt.Sprintf("%v", rv),
						Stack:     string(debug.Stack()),
						RequestID: requestID,
						Method:    r.Method,
						Path:      r.URL.Path,
					}
					if sl != nil {
						sl.Emit(panicRec)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error":      "internal server error",
						"request_id": requestID,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDEnsure is like RequestID but guaranteed to produce a non-empty ID.
// It's useful for the PanicRecovery middleware to always have a request ID.
func RequestIDEnsure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		if !strings.Contains(w.Header().Get("X-Request-ID"), requestID) {
			w.Header().Set("X-Request-ID", requestID)
		}
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
