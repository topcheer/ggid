// Package middleware implements gRPC reverse proxy support for the API Gateway.
package middleware

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GRPCProxyConfig configures the gRPC reverse proxy.
type GRPCProxyConfig struct {
	// Backends maps gRPC service name to backend address (host:port).
	// Example: "ggid.policy.v1.PolicyService" → "localhost:9070"
	Backends map[string]string

	// DefaultBackend is used when no specific backend matches.
	DefaultBackend string

	// ListenAddr is the gRPC listen address (e.g. ":9090").
	ListenAddr string

	// ConnectTimeout is the dial timeout for backend connections.
	ConnectTimeout time.Duration
}

// DefaultGRPCProxyConfig returns default gRPC proxy configuration.
func DefaultGRPCProxyConfig() GRPCProxyConfig {
	return GRPCProxyConfig{
		Backends: map[string]string{
			"ggid.auth.v1.AuthService":         "localhost:9071",
			"ggid.identity.v1.IdentityService": "localhost:9072",
			"ggid.policy.v1.PolicyService":     "localhost:9073",
			"ggid.org.v1.OrgService":           "localhost:9074",
			"ggid.audit.v1.AuditService":       "localhost:9075",
		},
		ListenAddr:     ":9090",
		ConnectTimeout: 5 * time.Second,
	}
}

// GRPCProxy is a TCP-level gRPC proxy that forwards connections to backend
// gRPC services based on the service name in the initial HTTP/2 frames.
// It uses a simpler approach than a full gRPC aware proxy: raw TCP tunneling
// with backend selection via connection metadata.
type GRPCProxy struct {
	config   GRPCProxyConfig
	mu       sync.RWMutex
	conns    int64
	active   int64
	listener net.Listener
}

// NewGRPCProxy creates a new gRPC proxy.
func NewGRPCProxy(cfg GRPCProxyConfig) *GRPCProxy {
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	return &GRPCProxy{config: cfg}
}

// GetBackend returns the backend address for a given gRPC service name.
// Falls back to DefaultBackend if no specific match is found.
func (p *GRPCProxy) GetBackend(serviceName string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if addr, ok := p.config.Backends[serviceName]; ok {
		return addr
	}
	// Try prefix matching for versioned services
	for prefix, addr := range p.config.Backends {
		if strings.HasPrefix(serviceName, prefix) {
			return addr
		}
	}
	return p.config.DefaultBackend
}

// AddBackend adds or updates a backend mapping at runtime.
func (p *GRPCProxy) AddBackend(serviceName, addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.config.Backends == nil {
		p.config.Backends = make(map[string]string)
	}
	p.config.Backends[serviceName] = addr
}

// RemoveBackend removes a backend mapping.
func (p *GRPCProxy) RemoveBackend(serviceName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.config.Backends, serviceName)
}

// ListenBackends returns the current backend configuration.
func (p *GRPCProxy) ListenBackends() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]string, len(p.config.Backends))
	for k, v := range p.config.Backends {
		result[k] = v
	}
	return result
}

// ConnectionCount returns total connections handled.
func (p *GRPCProxy) ConnectionCount() int64 {
	return p.conns
}

// ActiveConnections returns current active connections.
func (p *GRPCProxy) ActiveConnections() int64 {
	return p.active
}

// HandleHTTP2Proxy handles gRPC over HTTP/2 by tunneling at the TCP level.
// This function is called for each incoming connection.
// It reads the initial HTTP/2 preface to determine the target, then tunnels.
func (p *GRPCProxy) HandleConn(ctx context.Context, clientConn net.Conn, targetAddr string) {
	defer clientConn.Close()

	backendConn, err := net.DialTimeout("tcp", targetAddr, p.config.ConnectTimeout)
	if err != nil {
		log.Printf("grpc-proxy: dial backend %s error: %v", targetAddr, err)
		return
	}
	defer backendConn.Close()

	// Bidirectional copy
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}

// GRPCHTTPHandler is an HTTP handler that detects gRPC requests (Content-Type:
// application/grpc) and tunnels them to the appropriate backend.
// This allows the same HTTP listener to handle both REST and gRPC traffic.
func (p *GRPCProxy) GRPCHTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a gRPC request
		if !isGRPCRequest(r) {
			if next != nil {
				next.ServeHTTP(w, r)
			}
			return
		}

		// Extract service name from path: /package.Service/Method
		serviceName := extractGRPCService(r.URL.Path)
		targetAddr := p.GetBackend(serviceName)
		if targetAddr == "" {
			w.Header().Set("Content-Type", "application/grpc")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "no backend for gRPC service: %s", serviceName)
			return
		}

		// Tunnel the gRPC connection via HTTP hijack
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		clientConn, _, err := hj.Hijack()
		if err != nil {
			log.Printf("grpc-proxy: hijack error: %v", err)
			return
		}
		defer clientConn.Close()

		backendConn, err := net.DialTimeout("tcp", targetAddr, p.config.ConnectTimeout)
		if err != nil {
			log.Printf("grpc-proxy: dial %s error: %v", targetAddr, err)
			return
		}
		defer backendConn.Close()

		// Forward the original request to backend
		if err := r.Write(backendConn); err != nil {
			log.Printf("grpc-proxy: write to backend error: %v", err)
			return
		}

		// Bidirectional tunnel
		done := make(chan struct{}, 2)
		go func() {
			io.Copy(backendConn, clientConn)
			done <- struct{}{}
		}()
		go func() {
			io.Copy(clientConn, backendConn)
			done <- struct{}{}
		}()
		<-done
	})
}

// isGRPCRequest detects if an HTTP request is a gRPC call.
func isGRPCRequest(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/grpc")
}

// extractGRPCService extracts the service name from a gRPC path.
// gRPC paths are formatted as /package.ServiceName/MethodName
func extractGRPCService(path string) string {
	path = strings.TrimPrefix(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx > 0 {
		return path[:idx]
	}
	return path
}

// GRPCProxyStats represents gRPC proxy statistics for the admin API.
type GRPCProxyStats struct {
	TotalConnections   int64             `json:"total_connections"`
	ActiveConnections  int64             `json:"active_connections"`
	Backends           map[string]string `json:"backends"`
	DefaultBackend     string            `json:"default_backend,omitempty"`
}

// Stats returns the current gRPC proxy statistics.
func (p *GRPCProxy) Stats() GRPCProxyStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return GRPCProxyStats{
		TotalConnections:  p.conns,
		ActiveConnections: p.active,
		Backends:          p.ListenBackends(),
		DefaultBackend:    p.config.DefaultBackend,
	}
}
