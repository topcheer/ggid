package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Gzip compresses responses for clients that accept gzip encoding.
// It skips already-compressed content types and small responses (<512 bytes).
func Gzip(next http.Handler) http.Handler {
	pool := &sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
			return w
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip gzip for API endpoints — Next.js rewrite proxy doesn't handle
		// compressed responses correctly, causing empty body in browser fetch()
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/oauth/") || strings.HasPrefix(r.URL.Path, "/saml/") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			pool:           pool,
		}
		defer gw.Close()

		next.ServeHTTP(gw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	pool   *sync.Pool
	writer *gzip.Writer
	wrote  bool
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	// Check content type — skip compression for binary/image content
	ct := g.Header().Get("Content-Type")
	if shouldSkipCompression(ct) {
		return g.ResponseWriter.Write(data)
	}

	if !g.wrote {
		g.wrote = true
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
		g.writer = g.pool.Get().(*gzip.Writer)
		g.writer.Reset(g.ResponseWriter)
	}

	return g.writer.Write(data)
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	ct := g.Header().Get("Content-Type")
	if shouldSkipCompression(ct) {
		g.ResponseWriter.WriteHeader(code)
		return
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Close() {
	if g.writer != nil {
		g.writer.Flush()
		g.pool.Put(g.writer)
	}
}

// compressWriter is an alias for gzipResponseWriter for backward compatibility.
type compressWriter = gzipResponseWriter

// GzipBrotli is an alias for Gzip for backward compatibility.
func GzipBrotli(next http.Handler) http.Handler { return Gzip(next) }

// shouldSkipCompression returns true for content types that should not be compressed.
func shouldSkipCompression(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" {
		return false
	}
	skipPrefixes := []string{
		"image/", "video/", "audio/",
		"application/zip", "application/gzip", "application/x-gzip",
		"application/x-brotli", "application/x-bzip2",
		"application/x-7z-compressed", "application/x-rar-compressed",
		"application/pdf", "application/octet-stream",
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}
