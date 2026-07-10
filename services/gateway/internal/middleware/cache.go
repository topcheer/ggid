package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Cache provides simple in-memory response caching with ETag support.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	body       []byte
	header     http.Header
	etag       string
	expiresAt  time.Time
	statusCode int
}

// NewCache creates a response cache with the given TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Middleware caches GET responses with ETag/If-None-Match support.
func (c *Cache) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		key := cacheKey(r)

		// Check If-None-Match header
		if etag := r.Header.Get("If-None-Match"); etag != "" {
			c.mu.RLock()
			entry, ok := c.entries[key]
			c.mu.RUnlock()
			if ok && entry.etag == etag && time.Now().Before(entry.expiresAt) {
				w.Header().Set("ETag", entry.etag)
				w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(c.ttl.Seconds())))
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Check cache hit
		c.mu.RLock()
		entry, ok := c.entries[key]
		c.mu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) {
			// Serve from cache
			for k, v := range entry.header {
				w.Header()[k] = v
			}
			w.Header().Set("ETag", entry.etag)
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(c.ttl.Seconds())))
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.statusCode)
			w.Write(entry.body)
			return
		}

		// Cache miss — capture response
		cw := &cacheResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}, header: w.Header().Clone(), statusCode: 200}
		next.ServeHTTP(cw, r)

		// Only cache successful responses
		if cw.statusCode >= 200 && cw.statusCode < 300 && cw.buf.Len() > 0 {
			etag := generateETag(cw.buf.Bytes())
			c.mu.Lock()
			c.entries[key] = &cacheEntry{
				body:       cw.buf.Bytes(),
				header:     cw.header,
				etag:       etag,
				expiresAt:  time.Now().Add(c.ttl),
				statusCode: cw.statusCode,
			}
			c.mu.Unlock()
			// Set ETag on the real response
			w.Header().Set("ETag", etag)
		}
	})
}

// Invalidate clears all cached entries.
func (c *Cache) Invalidate() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

func cacheKey(r *http.Request) string {
	return r.Method + ":" + r.URL.RequestURI()
}

func generateETag(data []byte) string {
	h := sha256.Sum256(data)
	return `"` + hex.EncodeToString(h[:8]) + `"`
}

type cacheResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	header     http.Header
	statusCode int
}

func (w *cacheResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheResponseWriter) Write(data []byte) (int, error) {
	w.buf.Write(data)
	return w.ResponseWriter.Write(data)
}
