// Package router implements the HTTP reverse proxy router for the API Gateway.
package router

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// publicPaths are paths that skip JWT verification.
var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/password/forgot",
	"/api/v1/auth/password/reset",
	"/oauth/",
	"/saml/",
	"/.well-known/",
}

// Gateway is the API Gateway HTTP handler.
type Gateway struct {
	cfg      *config.Config
	jwks     *middleware.JWKSClient
	proxies  map[string]*httputil.ReverseProxy
	mu       sync.RWMutex
}

// New creates a new API Gateway handler.
func New(cfg *config.Config, jwks *middleware.JWKSClient) *Gateway {
	gw := &Gateway{
		cfg:     cfg,
		jwks:    jwks,
		proxies: make(map[string]*httputil.ReverseProxy),
	}
	gw.buildProxies()
	return gw
}

func (gw *Gateway) buildProxies() {
	for prefix, backendURL := range gw.cfg.Routes {
		parsed, err := url.Parse(backendURL)
		if err != nil {
			log.Printf("invalid backend URL %s: %v", backendURL, err)
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(parsed)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Forward resolved identity headers to the backend service
			if requestID, ok := req.Context().Value(middleware.RequestIDKey).(string); ok {
				req.Header.Set("X-Request-ID", requestID)
			}
			if userID, ok := middleware.UserIDFromRequest(req); ok {
				req.Header.Set("X-User-ID", userID.String())
			}
			if tenantID, ok := middleware.TenantIDFromRequest(req); ok {
				req.Header.Set("X-Tenant-ID", tenantID)
				// Also inject as query param for backend services that read tenant_id from URL
				q := req.URL.Query()
				if q.Get("tenant_id") == "" {
					q.Set("tenant_id", tenantID)
					req.URL.RawQuery = q.Encode()
				}
			}
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s%s: %v", parsed.Host, r.URL.Path, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "backend service unavailable"})
		}
		gw.proxies[prefix] = proxy
	}
}

// ServeHTTP routes the request to the appropriate backend service.
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// JWKS endpoint
	if r.URL.Path == "/.well-known/jwks.json" {
		gw.jwks.JWKSHandler()(w, r)
		return
	}

	// Find matching backend by longest prefix
	backend := gw.matchBackend(r.URL.Path)
	if backend == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no route for this path"})
		return
	}

	backend.ServeHTTP(w, r)
}

func (gw *Gateway) matchBackend(path string) *httputil.ReverseProxy {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	var bestMatch string
	for prefix := range gw.proxies {
		if strings.HasPrefix(path, prefix) {
			if len(prefix) > len(bestMatch) {
				bestMatch = prefix
			}
		}
	}
	if bestMatch == "" {
		return nil
	}
	return gw.proxies[bestMatch]
}

// Handler returns an http.Handler with all middleware applied in the correct order.
// Public paths (login, register, healthz, .well-known) skip JWT verification.
// All other paths require a valid JWT Bearer token.
func (gw *Gateway) Handler() http.Handler {
	// Inner handler: JWT enforcement + gateway routing
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a public path
		isPublic := false
		for _, pp := range publicPaths {
			if strings.HasPrefix(r.URL.Path, pp) {
				isPublic = true
				break
		}
		}
		// Health check and JWKS are always public
		if r.URL.Path == "/healthz" || r.URL.Path == "/.well-known/jwks.json" {
			isPublic = true
		}

		if isPublic {
			// Public path: no JWT required, but still validate if token present
			jwtMW := middleware.JWTAuth(gw.jwks, false, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			jwtMW(gw).ServeHTTP(w, r)
		} else {
			// Protected path: JWT required
			jwtMW := middleware.JWTAuth(gw.jwks, true, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			jwtMW(gw).ServeHTTP(w, r)
		}
	})

	// Apply outer middleware: CORS → RequestID → Logging → TenantResolver → inner
	handler := middleware.TenantResolver(gw.cfg.DomainSuffix)(inner)
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)

	return handler
}

// PrintRoutes logs the configured routes at startup.
func (gw *Gateway) PrintRoutes() {
	log.Println("API Gateway routes:")
	for prefix, backend := range gw.cfg.Routes {
		log.Printf("  %s -> %s", prefix, backend)
	}
}
