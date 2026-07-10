package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// ResponseCacheConfig configures the response caching middleware.
type ResponseCacheConfig struct {
	TTL             time.Duration // how long to cache responses
	MaxBodySize     int           // don't cache bodies larger than this (bytes)
	EnabledMethods  map[string]bool
}

// DefaultResponseCacheConfig returns sensible defaults.
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		TTL:         30 * time.Second,
		MaxBodySize: 64 * 1024, // 64KB
		EnabledMethods: map[string]bool{
			http.MethodGet: true,
			http.MethodHead: true,
		},
	}
}

type rcCachedResponse struct {
	status      int
	headers     http.Header
	body        []byte
	cachedAt    time.Time
	expiresAt   time.Time
	etag        string
	lastModified string
}

// ResponseCache provides an in-memory cache for GET responses.
// Supports ETag and Last-Modified conditional requests (304 Not Modified).
type ResponseCache struct {
	cfg    ResponseCacheConfig
	mu     sync.RWMutex
	cache  map[string]*rcCachedResponse
}

// NewResponseCache creates a new response cache.
func NewResponseCache(cfg ResponseCacheConfig) *ResponseCache {
	rc := &ResponseCache{
		cfg:   cfg,
		cache: make(map[string]*rcCachedResponse),
	}
	go rc.cleanup()
	return rc
}

// Middleware returns HTTP middleware that caches GET responses.
func (rc *ResponseCache) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache enabled methods
		if !rc.cfg.EnabledMethods[r.Method] {
			next.ServeHTTP(w, r)
			return
		}

		// Skip cache-control: no-cache
		if r.Header.Get("Cache-Control") == "no-cache" {
			next.ServeHTTP(w, r)
			return
		}

		key := rcCacheKey(r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-Tenant-ID"))

		// Check conditional request headers
		if cached := rc.get(key); cached != nil {
			// Check If-None-Match (ETag)
			if etag := r.Header.Get("If-None-Match"); etag != "" && etag == cached.etag {
				w.Header().Set("ETag", cached.etag)
				w.Header().Set("Last-Modified", cached.lastModified)
				w.WriteHeader(http.StatusNotModified)
				return
			}
			// Check If-Modified-Since
			if ims := r.Header.Get("If-Modified-Since"); ims != "" && ims == cached.lastModified {
				w.Header().Set("ETag", cached.etag)
				w.Header().Set("Last-Modified", cached.lastModified)
				w.WriteHeader(http.StatusNotModified)
				return
			}
			// Serve cached response
			for k, v := range cached.headers {
				for _, val := range v {
					w.Header().Add(k, val)
				}
			}
			w.Header().Set("ETag", cached.etag)
			w.Header().Set("Last-Modified", cached.lastModified)
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(cached.status)
			w.Write(cached.body)
			return
		}

		// Capture response
		cw := &rcCaptureWriter{ResponseWriter: w, header: make(http.Header)}
		next.ServeHTTP(cw, r)

		// Only cache successful responses with acceptable body size
		if cw.status >= 200 && cw.status < 300 && len(cw.body) <= rc.cfg.MaxBodySize {
			etag := rcGenerateETag(cw.body)
			lastMod := time.Now().UTC().Format(http.TimeFormat)
			rc.put(key, &rcCachedResponse{
				status:       cw.status,
				headers:      cw.header,
				body:         cw.body,
				cachedAt:     time.Now(),
				expiresAt:    time.Now().Add(rc.cfg.TTL),
				etag:         etag,
				lastModified: lastMod,
			})
			// Set cache headers on the original response
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", lastMod)
			w.Header().Set("X-Cache", "MISS")
		}
	})
}

func (rc *ResponseCache) get(key string) *rcCachedResponse {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	c, ok := rc.cache[key]
	if !ok {
		return nil
	}
	if time.Now().After(c.expiresAt) {
		return nil
	}
	return c
}

func (rc *ResponseCache) put(key string, resp *rcCachedResponse) {
	rc.mu.Lock()
	rc.cache[key] = resp
	rc.mu.Unlock()
}

// Invalidate removes a cached entry by key pattern.
func (rc *ResponseCache) Invalidate(pathPrefix string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for k := range rc.cache {
		if len(k) > len(pathPrefix) && k[:len(pathPrefix)] == pathPrefix {
			delete(rc.cache, k)
		}
	}
}

// Clear removes all cached entries.
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	rc.cache = make(map[string]*rcCachedResponse)
	rc.mu.Unlock()
}

func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(rc.cfg.TTL)
	defer ticker.Stop()
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for k, v := range rc.cache {
			if now.After(v.expiresAt) {
				delete(rc.cache, k)
			}
		}
		rc.mu.Unlock()
	}
}

// rcCaptureWriter captures the response body and status for caching.
type rcCaptureWriter struct {
	http.ResponseWriter
	status int
	header http.Header
	body   []byte
}

func (cw *rcCaptureWriter) WriteHeader(code int) {
	cw.status = code
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *rcCaptureWriter) Write(b []byte) (int, error) {
	cw.body = append(cw.body, b...)
	return cw.ResponseWriter.Write(b)
}

func (cw *rcCaptureWriter) Header() http.Header {
	if len(cw.header) == 0 {
		cw.header = cw.ResponseWriter.Header().Clone()
	}
	return cw.ResponseWriter.Header()
}

// --- helpers ---

func rcCacheKey(method, path, query, tenantID string) string {
	return method + "|" + path + "|" + query + "|" + tenantID
}

func rcGenerateETag(body []byte) string {
	h := sha256.Sum256(body)
	return `"` + hex.EncodeToString(h[:8]) + `"`
}
