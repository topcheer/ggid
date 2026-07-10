package middleware

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig configures the retry middleware.
type RetryConfig struct {
	MaxAttempts   int           // max retry attempts (default 3)
	InitialDelay  time.Duration // initial backoff delay (default 100ms)
	MaxDelay      time.Duration // max backoff delay (default 2s)
	RetryableStatus []int       // status codes to retry (default 502,503,504)
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        2 * time.Second,
		RetryableStatus: []int{502, 503, 504},
	}
}

// isRetryableMethod checks if the HTTP method is idempotent.
func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// isRetryableStatus checks if the status code is in the retryable list.
func (cfg *RetryConfig) isRetryableStatus(status int) bool {
	for _, s := range cfg.RetryableStatus {
		if s == status {
			return true
		}
	}
	return false
}

// backoffWithJitter returns exponential backoff with jitter.
func backoffWithJitter(attempt int, initial, max time.Duration) time.Duration {
	backoff := initial * time.Duration(1<<uint(attempt))
	if backoff > max {
		backoff = max
	}
	jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
	return backoff/2 + jitter
}

// retryResponseWriter captures the status code without writing to the real ResponseWriter.
// This allows the middleware to swallow retried responses.
type retryResponseWriter struct {
	header http.Header
	status int
	body   []byte
}

func newRetryResponseWriter() *retryResponseWriter {
	return &retryResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (w *retryResponseWriter) Header() http.Header { return w.header }
func (w *retryResponseWriter) WriteHeader(code int) { w.status = code }
func (w *retryResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

// RetryMiddleware retries idempotent requests on 5xx errors with exponential backoff + jitter.
func RetryMiddleware(cfg *RetryConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 100 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 2 * time.Second
	}
	if len(cfg.RetryableStatus) == 0 {
		cfg.RetryableStatus = []int{502, 503, 504}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only retry idempotent methods
			if !isRetryableMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			var lastStatus int
			rw := newRetryResponseWriter()
			for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
				if attempt > 0 {
					delay := backoffWithJitter(attempt-1, cfg.InitialDelay, cfg.MaxDelay)
					select {
					case <-time.After(delay):
					case <-r.Context().Done():
						w.WriteHeader(http.StatusRequestTimeout)
						return
					}
				}

				rw = newRetryResponseWriter()
				next.ServeHTTP(rw, r)
				lastStatus = rw.status

				// If not retryable, write response and return
				if !cfg.isRetryableStatus(rw.status) {
					// Copy headers
					for k, v := range rw.header {
						w.Header()[k] = v
					}
					w.WriteHeader(rw.status)
					w.Write(rw.body)
					if attempt > 0 {
						w.Header().Set("X-Retry-Count", strconv.Itoa(attempt))
					}
					return
				}
			}

			// Exhausted retries — write last response.
			if rw != nil {
				for k, v := range rw.header {
					w.Header()[k] = v
				}
			}
			w.Header().Set("X-Retry-Count", strconv.Itoa(cfg.MaxAttempts-1))
			w.WriteHeader(lastStatus)
			if rw != nil {
				w.Write(rw.body)
			}
		})
	}
}
