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
	reloadFunc    ReloadFunc
	routeVersion  int64
	stats         *middleware.StatsCollector
	graphql       *middleware.GraphQLResolver
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

	// --- Admin API ---
	if r.URL.Path == "/api/v1/admin/routes" && r.Method == http.MethodGet {
		gw.handleAdminRoutes(w, r)
		return
	}
	if r.URL.Path == "/api/v1/admin/stats" && r.Method == http.MethodGet {
		gw.handleAdminStats(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/routes") && strings.HasSuffix(r.URL.Path, "/toggle") && r.Method == http.MethodPost {
		gw.handleAdminToggleRoute(w, r)
		return
	}

	// --- Gateway management API ---
	if r.URL.Path == "/api/v1/gateway/routes" && r.Method == http.MethodGet {
		gw.handleGetRoutes(w, r)
		return
	}
	if r.URL.Path == "/api/v1/gateway/routes/reload" && r.Method == http.MethodPost {
		gw.handleReloadRoutes(w, r)
		return
	}
	if r.URL.Path == "/api/v1/gateway/middleware" && r.Method == http.MethodGet {
		middleware.MiddlewareChainHandler().ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/api/v1/gateway/stats" && r.Method == http.MethodGet {
		if gw.stats != nil {
			gw.stats.StatsHandler().ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "stats not configured"})
		}
		return
	}
	if r.URL.Path == "/graphql" && r.Method == http.MethodPost {
		if gw.graphql != nil {
			gw.graphql.GraphQLHandler().ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "GraphQL not configured"})
		}
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

	// Apply outer middleware: PanicRecovery → CORS → RequestID → StructuredLogging → TenantResolver → inner
	logger := middleware.NewStructuredLogger("ggid-gateway")
	handler := middleware.TenantResolver(gw.cfg.DomainSuffix)(inner)
	handler = middleware.RequestLogger(logger)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)
	handler = middleware.PanicRecovery(logger)(handler)

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

// --- Gateway Admin API ---

// RouteInfo describes a single route in the gateway.
type RouteInfo struct {
	Prefix      string `json:"prefix"`
	Backend     string `json:"backend"`
	HasTimeout  bool   `json:"has_timeout"`
	ReadTimeout string `json:"read_timeout,omitempty"`
}

// RoutesResponse is the response for GET /api/v1/gateway/routes.
type RoutesResponse struct {
	Routes  []RouteInfo `json:"routes"`
	Version int64       `json:"version"`
}

// ReloadResponse is the response for POST /api/v1/gateway/routes/reload.
type ReloadResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Version int64  `json:"version"`
}

// ReloadFunc is a callback to reload the configuration at runtime.
// If nil, reload returns an error.
type ReloadFunc func() (*config.Config, error)

// handleGetRoutes returns the current route table as JSON.
func (gw *Gateway) handleGetRoutes(w http.ResponseWriter, _ *http.Request) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	routes := make([]RouteInfo, 0, len(gw.cfg.Routes))
	for prefix, backend := range gw.cfg.Routes {
		ri := RouteInfo{
			Prefix:  prefix,
			Backend: backend,
		}
		if to, ok := gw.timeouts[prefix]; ok && to > 0 {
			ri.HasTimeout = true
			ri.ReadTimeout = to.String()
		}
		routes = append(routes, ri)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RoutesResponse{
		Routes:  routes,
		Version: gw.routeVersion,
	})
}

// handleReloadRoutes triggers a route reload from the config source.
func (gw *Gateway) handleReloadRoutes(w http.ResponseWriter, _ *http.Request) {
	if gw.reloadFunc == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(ReloadResponse{
			Status:  "error",
			Message: "reload not configured",
		})
		return
	}

	newCfg, err := gw.reloadFunc()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ReloadResponse{
			Status:  "error",
			Message: "reload failed: " + err.Error(),
		})
		return
	}

	gw.mu.Lock()
	gw.cfg = newCfg
	gw.proxies = make(map[string]*httputil.ReverseProxy)
	gw.timeouts = make(map[string]time.Duration)
	gw.buildProxiesLocked()
	gw.buildHealthChecker()
	gw.routeVersion++
	gw.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ReloadResponse{
		Status:  "ok",
		Message: "routes reloaded",
		Version: gw.routeVersion,
	})
}

// SetReloadFunc sets the callback used by the reload endpoint.
func (gw *Gateway) SetReloadFunc(fn ReloadFunc) {
	gw.reloadFunc = fn
}

// --- Admin API Handlers ---

// AdminRouteInfo describes a route for the admin API, including enabled state.
type AdminRouteInfo struct {
	Prefix   string `json:"prefix"`
	Backend  string `json:"backend"`
	Enabled  bool   `json:"enabled"`
	Timeout  string `json:"timeout,omitempty"`
}

// AdminStatsResponse holds per-backend statistics.
type AdminStatsResponse struct {
	Backends map[string]*BackendStats `json:"backends"`
}

// BackendStats holds statistics for a single backend.
type BackendStats struct {
	RequestCount   int64   `json:"request_count"`
	ErrorCount     int64   `json:"error_count"`
	ErrorRate      float64 `json:"error_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
}

// handleAdminRoutes returns all route configurations with enabled state.
func (gw *Gateway) handleAdminRoutes(w http.ResponseWriter, _ *http.Request) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	routes := make([]AdminRouteInfo, 0, len(gw.cfg.Routes))
	for prefix, backend := range gw.cfg.Routes {
		ri := AdminRouteInfo{
			Prefix:  prefix,
			Backend: backend,
			Enabled: true,
		}
		if _, exists := gw.proxies[prefix]; !exists {
			ri.Enabled = false
		}
		if to, ok := gw.timeouts[prefix]; ok && to > 0 {
			ri.Timeout = to.String()
		}
		routes = append(routes, ri)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"routes":  routes,
		"version": gw.routeVersion,
	})
}

// handleAdminStats returns aggregated per-backend statistics.
func (gw *Gateway) handleAdminStats(w http.ResponseWriter, _ *http.Request) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	resp := &AdminStatsResponse{
		Backends: make(map[string]*BackendStats),
	}

	for prefix := range gw.cfg.Routes {
		resp.Backends[prefix] = &BackendStats{
			RequestCount: 0,
			ErrorCount:   0,
			ErrorRate:    0,
			AvgLatencyMs: 0,
			P99LatencyMs: 0,
		}
	}

	// If stats collector is configured, merge real data
	if gw.stats != nil {
		snap := gw.stats.Snapshot()
		for prefix, rs := range snap.Routes {
			if bs, ok := resp.Backends[prefix]; ok {
				bs.RequestCount = int64(rs.Requests)
				bs.ErrorCount = int64(rs.Errors)
				if rs.Requests > 0 {
					bs.ErrorRate = float64(rs.Errors) / float64(rs.Requests)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleAdminToggleRoute enables or disables a route by prefix.
func (gw *Gateway) handleAdminToggleRoute(w http.ResponseWriter, r *http.Request) {
	// Extract route prefix from URL: /api/v1/admin/routes/{prefix}/toggle
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/routes/")
	prefix := strings.TrimSuffix(path, "/toggle")
	if prefix == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "route prefix required"})
		return
	}

	gw.mu.Lock()
	defer gw.mu.Unlock()

	backendURL, exists := gw.cfg.Routes[prefix]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "route not found"})
		return
	}

	// Check if currently enabled
	if _, proxied := gw.proxies[prefix]; proxied {
		// Disable: remove proxy
		delete(gw.proxies, prefix)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"prefix":  prefix,
			"enabled": false,
		})
	} else {
		// Enable: recreate proxy
		parsed, err := url.Parse(backendURL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid backend URL"})
			return
		}
		gw.proxies[prefix] = httputil.NewSingleHostReverseProxy(parsed)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"prefix":  prefix,
			"enabled": true,
		})
	}
}

// buildProxiesLocked builds proxies without acquiring the write lock.
// Caller must hold gw.mu.
func (gw *Gateway) buildProxiesLocked() {
	for prefix, backendURL := range gw.cfg.Routes {
		parsed, err := url.Parse(backendURL)
		if err != nil {
			log.Printf("invalid backend URL %s: %v", backendURL, err)
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(parsed)

		to := gw.cfg.GetRouteTimeout(prefix)
		proxy.Transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
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
			if requestID, ok := req.Context().Value(middleware.RequestIDKey).(string); ok {
				req.Header.Set("X-Request-ID", requestID)
			}
			if userID, ok := middleware.UserIDFromRequest(req); ok {
				req.Header.Set("X-User-ID", userID.String())
			}
			if tenantID, ok := middleware.TenantIDFromRequest(req); ok {
				req.Header.Set("X-Tenant-ID", tenantID)
				q := req.URL.Query()
				if q.Get("tenant_id") == "" {
					q.Set("tenant_id", tenantID)
					req.URL.RawQuery = q.Encode()
				}
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
		if to.Read > 0 {
			gw.timeouts[prefix] = to.Read
		}
	}
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
