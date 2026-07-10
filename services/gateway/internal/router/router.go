// Package router implements the HTTP reverse proxy router for the API Gateway.
package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/healthcheck"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// publicPaths are paths that skip JWT verification.
var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/password/forgot",
	"/api/v1/auth/password/reset",
	"/api/v1/auth/social/",
	"/oauth/",
	"/saml/",
	"/.well-known/",
	"/docs",
	"/api-docs",
	"/login",
	"/register",
	"/forgot-password",
}

// Gateway is the API Gateway HTTP handler.
type Gateway struct {
	cfg           *config.Config
	jwks          *middleware.JWKSClient
	proxies       map[string]*httputil.ReverseProxy
	timeouts      map[string]time.Duration
	healthChecker *healthcheck.Checker
	mu            sync.RWMutex
}

// New creates a new API Gateway handler.
func New(cfg *config.Config, jwks *middleware.JWKSClient) *Gateway {
	gw := &Gateway{
		cfg:      cfg,
		jwks:     jwks,
		proxies:  make(map[string]*httputil.ReverseProxy),
		timeouts: make(map[string]time.Duration),
	}
	gw.buildProxies()
	gw.buildHealthChecker()
	return gw
}

// SetHealthChecker allows injecting a pre-configured health checker.
func (gw *Gateway) SetHealthChecker(hc *healthcheck.Checker) {
	gw.healthChecker = hc
}

func (gw *Gateway) buildHealthChecker() {
	services := make(map[string]string)
	for prefix, backendURL := range gw.cfg.Routes {
		// Convert route prefix to service name for health check
		name := strings.TrimPrefix(prefix, "/api/v1/")
		if name == "" {
			name = strings.TrimPrefix(prefix, "/")
		}
		services[name] = backendURL + "/healthz"
	}
	gw.healthChecker = healthcheck.NewChecker(services)
}

func (gw *Gateway) buildProxies() {
	for prefix, backendURL := range gw.cfg.Routes {
		parsed, err := url.Parse(backendURL)
		if err != nil {
			log.Printf("invalid backend URL %s: %v", backendURL, err)
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(parsed)

		// Optimized connection pooling Transport
		// Per-route timeout configuration
		to := gw.cfg.GetRouteTimeout(prefix)
		proxy.Transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     0, // unlimited
			IdleConnTimeout:     to.Idle,
			DialContext: (&net.Dialer{
				Timeout:   to.Dial,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

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
				// Inject as query param for GET requests
				q := req.URL.Query()
				if q.Get("tenant_id") == "" {
					q.Set("tenant_id", tenantID)
					req.URL.RawQuery = q.Encode()
				}
				// Inject into JSON body for POST/PUT/PATCH requests
				injectTenantIntoBody(req, tenantID)
			}
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s%s: %v", parsed.Host, r.URL.Path, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "backend service unavailable"})
		}
		gw.proxies[prefix] = proxy
		// Store per-route read timeout for context-based timeout in ServeHTTP
		if to.Read > 0 {
			gw.timeouts[prefix] = to.Read
		}
	}
}

// ServeHTTP routes the request to the appropriate backend service.
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// --- Liveness probe (Kubernetes: process alive, no backend check) ---
	if r.URL.Path == "/healthz/live" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
		return
	}

	// --- Readiness probe (Kubernetes: checks all backends) ---
	if r.URL.Path == "/healthz/ready" {
		if gw.healthChecker != nil {
			gw.healthChecker.ReadyHandler().ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		}
		return
	}

	// --- Basic health check ---
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	// Prometheus metrics
	if r.URL.Path == "/metrics" {
		middleware.MetricsHandler().ServeHTTP(w, r)
		return
	}

	// JWKS endpoint
	if r.URL.Path == "/.well-known/jwks.json" {
		gw.jwks.JWKSHandler()(w, r)
		return
	}

	// API documentation (Swagger UI)
	if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerHTML))
		return
	}

	// Hosted login page (served by Gateway — any app can redirect here)
	if r.URL.Path == "/login" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedLoginHTML))
		return
	}

	// Hosted registration page
	if r.URL.Path == "/register" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedRegisterHTML))
		return
	}

	// Password reset page
	if r.URL.Path == "/forgot-password" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedForgotPasswordHTML))
		return
	}

	// OpenAPI JSON spec
	if r.URL.Path == "/api-docs" {
		serveOpenAPISpec(w, r)
		return
	}

	// Find matching backend by longest prefix
	backend, prefix := gw.matchBackend(r.URL.Path)
	if backend == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no route for this path"})
		return
	}

	// Apply per-route timeout if configured
	if to, ok := gw.timeouts[prefix]; ok && to > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), to)
		defer cancel()
		r = r.WithContext(ctx)
	}
	backend.ServeHTTP(w, r)
}

// matchBackend returns the proxy and its prefix for the given path.
func (gw *Gateway) matchBackend(path string) (*httputil.ReverseProxy, string) {
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
		return nil, ""
	}
	return gw.proxies[bestMatch], bestMatch
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
		if r.URL.Path == "/healthz" || r.URL.Path == "/healthz/live" || r.URL.Path == "/healthz/ready" || r.URL.Path == "/.well-known/jwks.json" || r.URL.Path == "/metrics" {
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

// injectTenantIntoBody injects tenant_id into the JSON body of POST/PUT/PATCH requests.
// It only modifies flat JSON objects and preserves the original body if it's not JSON
// or already contains a tenant_id field.
func injectTenantIntoBody(req *http.Request, tenantID string) {
	if req.Body == nil || tenantID == "" {
		return
	}
	// Only modify JSON bodies for write methods
	if req.Method != http.MethodPost && req.Method != http.MethodPut && req.Method != http.MethodPatch {
		return
	}
	ct := req.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return
	}
	// Restore body if anything fails
	restore := func() {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var bodyMap map[string]any
	if json.Unmarshal(bodyBytes, &bodyMap) != nil {
		restore()
		return
	}
	// Skip if tenant_id already present
	if _, exists := bodyMap["tenant_id"]; exists {
		restore()
		return
	}

	bodyMap["tenant_id"] = tenantID
	newBody, err := json.Marshal(bodyMap)
	if err != nil {
		restore()
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
}

// PrintRoutes logs the configured routes at startup.
func (gw *Gateway) PrintRoutes() {
	log.Println("API Gateway routes:")
	for prefix, backend := range gw.cfg.Routes {
		log.Printf("  %s -> %s", prefix, backend)
	}
	log.Println("  /docs -> Swagger UI")
	log.Println("  /api-docs -> OpenAPI JSON spec")
}

// serveSwaggerUI writes the Swagger UI HTML page.
func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

// serveOpenAPISpec writes the OpenAPI 3.0 JSON spec.
func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}
