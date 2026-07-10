package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// compressibleContentTypes lists content types that benefit from gzip.
var compressibleContentTypes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"application/xml",
	"application/xhtml+xml",
	"image/svg+xml",
}

// skipCompressionPrefixes are response content types that are already compressed.
var skipCompressionPrefixes = []string{
	"image/",
	"video/",
	"audio/",
	"application/zip",
	"application/gzip",
	"application/x-gzip",
	"application/octet-stream",
}

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

func shouldSkipCompression(contentType string) bool {
	contentType = strings.ToLower(contentType)

	// SVG is compressible despite image/ prefix
	if strings.HasPrefix(contentType, "image/svg+xml") {
		return false
	}

	for _, prefix := range skipCompressionPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	// Check if it's a compressible type
	for _, ct := range compressibleContentTypes {
		if strings.HasPrefix(contentType, ct) {
			return false
		}
	}
	return true // skip unknown types
}
