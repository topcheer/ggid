package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

// RequestCoalescer deduplicates concurrent identical GET requests.
// When multiple requests arrive for the same URL while one is in-flight,
// they wait for the first response and share the result.
type RequestCoalescer struct {
	mu       sync.Mutex
	inflight map[string]*coalescedCall
	ttl      time.Duration // how long to cache successful responses
	cache    map[string]*cachedResponse
}

type coalescedCall struct {
	done   chan struct{}
	status int
	body   []byte
	header http.Header
	err    error
}

type cachedResponse struct {
	status  int
	body    []byte
	header  http.Header
	expires time.Time
}

// NewRequestCoalescer creates a coalescer with the given cache TTL.
func NewRequestCoalescer(ttl time.Duration) *RequestCoalescer {
	rc := &RequestCoalescer{
		inflight: make(map[string]*coalescedCall),
		cache:    make(map[string]*cachedResponse),
		ttl:      ttl,
	}
	if ttl > 0 {
		go rc.cleanupLoop()
	}
	return rc
}

func (rc *RequestCoalescer) cleanupLoop() {
	ticker := time.NewTicker(rc.ttl)
	defer ticker.Stop()
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for k, v := range rc.cache {
			if now.After(v.expires) {
				delete(rc.cache, k)
			}
		}
		rc.mu.Unlock()
	}
}

// coalesceKey generates a deduplication key from method + URL.
func coalesceKey(method, path, query string) string {
	return method + " " + path + "?" + query
}

// CoalesceMiddleware deduplicates concurrent identical GET requests.
// Only GET requests are coalesced; other methods pass through.
func CoalesceMiddleware(rc *RequestCoalescer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only coalesce safe GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := coalesceKey(r.Method, r.URL.Path, r.URL.RawQuery)

			// Check cache first
			rc.mu.Lock()
			if cached, ok := rc.cache[key]; ok && time.Now().Before(cached.expires) {
				rc.mu.Unlock()
				copyResponse(w, cached.status, cached.body, cached.header)
				return
			}

			// Check if there's already an in-flight request for this key
			if call, ok := rc.inflight[key]; ok {
				rc.mu.Unlock()
				// Wait for the in-flight call to complete
				<-call.done
				copyResponse(w, call.status, call.body, call.header)
				return
			}

			// Create a new coalesced call
			call := &coalescedCall{
				done:   make(chan struct{}),
				header: make(http.Header),
			}
			rc.inflight[key] = call
			rc.mu.Unlock()

			// Execute the request
			sr := &coalesceRecorder{ResponseWriter: w, status: http.StatusOK, header: call.header}
			next.ServeHTTP(sr, r)

			call.status = sr.status
			call.body = sr.body.Bytes()

			// Cache successful responses
			if rc.ttl > 0 && sr.status >= 200 && sr.status < 300 {
				rc.mu.Lock()
				rc.cache[key] = &cachedResponse{
					status:  sr.status,
					body:    call.body,
					header:  call.header,
					expires: time.Now().Add(rc.ttl),
				}
				rc.mu.Unlock()
			}

			// Signal waiting callers
			close(call.done)

			// Remove from inflight
			rc.mu.Lock()
			delete(rc.inflight, key)
			rc.mu.Unlock()
		})
	}
}

func copyResponse(w http.ResponseWriter, status int, body []byte, header http.Header) {
	for k, v := range header {
		w.Header()[k] = v
	}
	w.WriteHeader(status)
	w.Write(body)
}

// coalesceRecorder captures the response for coalescing.
type coalesceRecorder struct {
	http.ResponseWriter
	status int
	header http.Header
	body   bytes.Buffer
}

func (c *coalesceRecorder) Header() http.Header {
	if c.header != nil {
		return c.header
	}
	return c.ResponseWriter.Header()
}

func (c *coalesceRecorder) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *coalesceRecorder) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}

// Ensure io is imported
var _ = io.Discard
