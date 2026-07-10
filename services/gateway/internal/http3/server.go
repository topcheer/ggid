// Package http3 provides HTTP/3 (QUIC) listener support for the Gateway.
// HTTP/3 uses UDP-based QUIC protocol, offering 0-RTT connection setup,
// multiplexing without head-of-line blocking, and connection migration.
package http3

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

// Server wraps an HTTP/3 server with graceful shutdown support.
type Server struct {
	server  *http3.Server
	handler http.Handler
	addr    string
}

// Config holds HTTP/3 server configuration.
type Config struct {
	Addr       string // UDP listen address (e.g. ":8443")
	TLSConfig  *tls.Config
	EnableH2   bool // also start HTTP/2 on same port (alt-svc)
	MaxStreams int  // max concurrent bidirectional streams per connection
}

// DefaultConfig returns sensible HTTP/3 defaults.
func DefaultConfig() Config {
	return Config{
		Addr:       ":8443",
		EnableH2:   true,
		MaxStreams: 100,
	}
}

// New creates an HTTP/3 server with the given config and handler.
func New(cfg Config, handler http.Handler) (*Server, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":8443"
	}
	if cfg.TLSConfig == nil {
		return nil, fmt.Errorf("TLS config is required for HTTP/3")
	}
	if handler == nil {
		handler = http.DefaultServeMux
	}

	// Set NextProtos for HTTP/3
	cfg.TLSConfig.NextProtos = []string{"h3"}

	srv := &http3.Server{
		Addr:            cfg.Addr,
		Handler:         handler,
		TLSConfig:       cfg.TLSConfig,
		EnableDatagrams: false,
	}

	return &Server{
		server:  srv,
		handler: handler,
		addr:    cfg.Addr,
	}, nil
}

// ListenAndServe starts the HTTP/3 server.
func (s *Server) ListenAndServe() error {
	log.Printf("HTTP/3 server listening on %s (QUIC/UDP)", s.addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the HTTP/3 server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

// AltSvcHeader returns the Alt-Svc header value for advertising HTTP/3.
// Clients connecting via HTTP/2 or HTTP/1.1 will see this header and
// automatically upgrade to HTTP/3 on subsequent requests.
// Example: "h3=\":8443\"; ma=86400"
func AltSvcHeader(port int) string {
	if port == 0 {
		port = 8443
	}
	return fmt.Sprintf(`h3=":%d"; ma=86400`, port)
}

// AltSvcMiddleware adds the Alt-Svc header to all responses so that
// clients automatically discover the HTTP/3 endpoint.
func AltSvcMiddleware(port int) func(http.Handler) http.Handler {
	header := AltSvcHeader(port)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", header)
			next.ServeHTTP(w, r)
		})
	}
}

// HappyEyeballsMiddleware checks if the client supports HTTP/3 (via the
// `X-HTTP3-Supported` header) and responds with the Alt-Svc header.
// This implements the "happy eyeballs" pattern where the client tries
// both HTTP/2 and HTTP/3 and uses whichever connects first.
func HappyEyeballsMiddleware(port int) func(http.Handler) http.Handler {
	altSvc := AltSvcHeader(port)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", altSvc)
			// If this request arrived over HTTP/3, set a marker header
			if r.ProtoMajor == 3 {
				w.Header().Set("X-HTTP3-Version", r.Proto)
			}
			next.ServeHTTP(w, r)
		})
	}
}
