// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

// GzipBrotli compresses responses using the best available algorithm.
// If the client supports brotli (br), it is preferred over gzip because of
// better compression ratios.  The compression level is chosen automatically
// based on the response Content-Type:
//
//   - text/html, text/css, application/javascript → level 6 (max compression)
//   - application/json, application/xml             → level 4 (balanced)
//   - everything else                                → level 1 (speed)
//
// Already-compressed content types (images, video, etc.) are passed through.
func GzipBrotli(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ae := r.Header.Get("Accept-Encoding")
		if ae == "" {
			next.ServeHTTP(w, r)
			return
		}

		supportsBrotli := acceptEncoding(ae, "br")
		supportsGzip := acceptEncoding(ae, "gzip")
		if !supportsBrotli && !supportsGzip {
			next.ServeHTTP(w, r)
			return
		}

		cw := &compressWriter{
			ResponseWriter:   w,
			supportsBrotli:   supportsBrotli,
			supportsGzip:     supportsGzip,
			gzipPool:         gzipWriterPool,
			brotliPool:       brotliWriterPool,
		}
		defer cw.Close()
		next.ServeHTTP(cw, r)
	})
}

var (
	gzipWriterPool = newGzipPool()
	brotliWriterPool = newBrotliPool()
)

func acceptEncoding(headerVal, enc string) bool {
	for _, part := range strings.Split(headerVal, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, enc) {
			return true
		}
	}
	return false
}

func compressionLevelForType(contentType string) int {
	ct := strings.ToLower(contentType)
	// High compression for text-heavy content
	if strings.HasPrefix(ct, "text/html") ||
		strings.HasPrefix(ct, "text/css") ||
		strings.HasPrefix(ct, "application/javascript") ||
		strings.HasPrefix(ct, "application/xhtml+xml") {
		return 6
	}
	// Balanced for structured data
	if strings.HasPrefix(ct, "application/json") ||
		strings.HasPrefix(ct, "application/xml") ||
		strings.HasPrefix(ct, "text/plain") ||
		strings.HasPrefix(ct, "text/csv") {
		return 4
	}
	// Speed for everything else
	return 1
}

// compressWriter handles brotli/gzip encoding transparently.
type compressWriter struct {
	http.ResponseWriter
	supportsBrotli bool
	supportsGzip   bool
	gzipPool       *gzipSyncPool
	brotliPool     *brotliSyncPool
	writer         flushCloseWriter
	level          int
	wroteHeader    bool
	skip           bool
}

// flushCloseWriter is the minimum interface shared by gzip.Writer and brotli.Writer.
type flushCloseWriter interface {
	io.WriteCloser
	Flush() error
}

func (cw *compressWriter) Write(data []byte) (int, error) {
	if !cw.wroteHeader {
		cw.setupWriter()
	}
	if cw.skip {
		return cw.ResponseWriter.Write(data)
	}
	return cw.writer.Write(data)
}

func (cw *compressWriter) WriteHeader(code int) {
	cw.setupWriter()
	if !cw.skip {
		cw.ResponseWriter.WriteHeader(code)
	} else {
		cw.ResponseWriter.WriteHeader(code)
	}
}

func (cw *compressWriter) setupWriter() {
	if cw.wroteHeader {
		return
	}
	cw.wroteHeader = true

	ct := cw.Header().Get("Content-Type")
	if shouldSkipCompression(ct) {
		cw.skip = true
		return
	}

	cw.level = compressionLevelForType(ct)

	if cw.supportsBrotli {
		cw.Header().Set("Content-Encoding", "br")
		cw.Header().Del("Content-Length")
		bw := cw.brotliPool.Get(cw.level)
		bw.Reset(cw.ResponseWriter)
		cw.writer = &brotliFlushWriter{Writer: bw, pool: cw.brotliPool, level: cw.level}
	} else if cw.supportsGzip {
		cw.Header().Set("Content-Encoding", "gzip")
		cw.Header().Del("Content-Length")
		gw := cw.gzipPool.Get(cw.level)
		gw.Reset(cw.ResponseWriter)
		cw.writer = &gzipFlushWriter{Writer: gw, pool: cw.gzipPool, level: cw.level}
	} else {
		cw.skip = true
	}
}

func (cw *compressWriter) Close() {
	if cw.writer != nil {
		cw.writer.Flush()
		cw.writer.Close()
		cw.writer = nil
	}
}

// --- gzip pool wrapper ---

type gzipFlushWriter struct {
	*gzip.Writer
	pool  *gzipSyncPool
	level int
}

func (g *gzipFlushWriter) Close() error {
	g.Writer.Flush()
	g.pool.Put(g.level, g.Writer)
	return nil
}

// --- brotli pool wrapper ---

type brotliFlushWriter struct {
	*brotli.Writer
	pool  *brotliSyncPool
	level int
}

func (b *brotliFlushWriter) Close() error {
	b.Writer.Flush()
	b.pool.Put(b.level, b.Writer)
	return nil
}

// --- pools ---

type gzipSyncPool struct {
	pools map[int]*gzipPoolEntry
}

type gzipPoolEntry struct {
	free chan *gzip.Writer
}

func newGzipPool() *gzipSyncPool {
	return &gzipSyncPool{
		pools: map[int]*gzipPoolEntry{
			1: {free: make(chan *gzip.Writer, 32)},
			4: {free: make(chan *gzip.Writer, 32)},
			6: {free: make(chan *gzip.Writer, 32)},
		},
	}
}

func (p *gzipSyncPool) Get(level int) *gzip.Writer {
	entry := p.pools[level]
	if entry == nil {
		entry = p.pools[6] // fallback
	}
	select {
	case w := <-entry.free:
		return w
	default:
		w, _ := gzip.NewWriterLevel(io.Discard, level)
		return w
	}
}

func (p *gzipSyncPool) Put(level int, w *gzip.Writer) {
	entry := p.pools[level]
	if entry == nil {
		entry = p.pools[6]
	}
	select {
	case entry.free <- w:
	default:
	}
}

type brotliSyncPool struct {
	pools map[int]*brotliPoolEntry
}

type brotliPoolEntry struct {
	free chan *brotli.Writer
}

func newBrotliPool() *brotliSyncPool {
	return &brotliSyncPool{
		pools: map[int]*brotliPoolEntry{
			1: {free: make(chan *brotli.Writer, 32)},
			4: {free: make(chan *brotli.Writer, 32)},
			6: {free: make(chan *brotli.Writer, 32)},
		},
	}
}

func (p *brotliSyncPool) Get(level int) *brotli.Writer {
	entry := p.pools[level]
	if entry == nil {
		entry = p.pools[6]
	}
	select {
	case w := <-entry.free:
		return w
	default:
		return brotli.NewWriterLevel(io.Discard, level)
	}
}

func (p *brotliSyncPool) Put(level int, w *brotli.Writer) {
	entry := p.pools[level]
	if entry == nil {
		entry = p.pools[6]
	}
	select {
	case entry.free <- w:
	default:
	}
}

// parseContentEncoding parses Accept-Encoding quality values.
func parseContentEncoding(header string) map[string]float64 {
	result := make(map[string]float64)
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		tokens := strings.Split(part, ";")
		enc := strings.TrimSpace(tokens[0])
		q := 1.0
		for _, t := range tokens[1:] {
			t = strings.TrimSpace(t)
			if strings.HasPrefix(t, "q=") {
				fmt.Sscanf(t[2:], "%f", &q)
			}
		}
		result[strings.ToLower(enc)] = q
	}
	return result
}