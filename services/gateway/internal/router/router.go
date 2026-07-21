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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/posture"
	"github.com/ggid/ggid/pkg/sysconfig"
	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/healthcheck"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
	pkgmiddleware "github.com/ggid/ggid/pkg/middleware"
)

// publicPaths are paths that skip JWT verification.
var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/password/forgot",
	"/api/v1/auth/password/reset",
	"/api/v1/auth/password/strength",
	"/api/v1/auth/mfa/verify",
	"/api/v1/auth/mfa/challenge",
	"/api/v1/auth/mfa/radius/verify",
	"/api/v1/auth/mfa/yubikey/verify",
	"/api/v1/auth/mfa/methods",
	"/api/v1/auth/mfa/login",
	"/api/v1/auth/social/",
	"/api/v1/healthz",
	"/healthz",
	"/api/v1/identity/healthz",
	"/api/v1/auth/healthz",
	"/api/v1/oauth/healthz",
	"/api/v1/policy/healthz",
	"/api/v1/org/healthz",
	"/api/v1/audit/healthz",
	"/api/v1/system/initialized",
	"/api/v1/system/bootstrap",
	"/api/v1/system/status",
	"/api/v1/tenants/resolve",
	"/api/v1/dashboard",
	"/api/v1/health",
	"/api/v1/oauth/jwks",
	"/api/v1/oauth/.well-known/",
	"/api/v1/oauth/token",
	"/api/v1/oauth/authorize",
	"/api/v1/oauth/revoke",
	"/api/v1/oauth/introspect",
	"/api/v1/oauth/device",
	"/api/v1/oauth/register",
	"/api/v1/oauth/backchannel",
	"/api/v1/oauth/userinfo",
	"/api/v1/auth/saml/",
	"/oauth/",
	"/saml/",
	"/.well-known/",
	"/docs",
	"/api-docs",
	"/login",
	"/register",
	"/forgot-password",
	"/device",
}

// Gateway is the API Gateway HTTP handler.
type Gateway struct {
	cfg            *config.Config
	jwks           *middleware.JWKSClient
	proxies        map[string]*httputil.ReverseProxy
	timeouts       map[string]time.Duration
	healthChecker  *healthcheck.Checker
	rateLimiter    *middleware.TenantBucketLimiter
	multiDimLimiter *middleware.MultiDimRateLimiter
	postureEngine   *posture.Engine
	postureDropFn   func(ctx context.Context, tenantID, userID string, score int)
	reloadFunc     ReloadFunc
	routeVersion   int64
	stats          *middleware.StatsCollector
	graphql        *middleware.GraphQLResolver
	sessionMgr     *middleware.SessionManager
	sysconfigStore sysconfig.Store
	internalSecret []byte
	caeCheck       func(http.Handler) http.Handler
	appRouter      *ProtectedAppRouter
	circuitRegistry *middleware.CircuitRegistry
	mu             sync.RWMutex
}

// SetCAECheck injects the CAE (Continuous Access Evaluation) middleware.
func (gw *Gateway) SetCAECheck(cae func(http.Handler) http.Handler) {
	gw.caeCheck = cae
}

// SetMultiDimRateLimiter injects the 5-dimensional rate limiter.
func (gw *Gateway) SetMultiDimRateLimiter(rl *middleware.MultiDimRateLimiter) {
	gw.multiDimLimiter = rl
}

// New creates a new API Gateway handler.
func New(cfg *config.Config, jwks *middleware.JWKSClient) *Gateway {
	gw := &Gateway{
		cfg:         cfg,
		jwks:        jwks,
		proxies:        make(map[string]*httputil.ReverseProxy),
		timeouts:       make(map[string]time.Duration),
		rateLimiter:    middleware.NewTenantBucketLimiter(middleware.DefaultBucketRateLimitConfig()),
		circuitRegistry: middleware.NewCircuitRegistry(middleware.DefaultCircuitConfig()),
		appRouter:      NewProtectedAppRouter(),
	}
	gw.buildProxies()
	gw.buildHealthChecker()
	return gw
}

// SetHealthChecker allows injecting a pre-configured health checker.
func (gw *Gateway) SetHealthChecker(hc *healthcheck.Checker) {
	gw.healthChecker = hc
}

// SetSessionManager allows injecting a session manager for timeout enforcement.
func (gw *Gateway) SetSessionManager(sm *middleware.SessionManager) {
	gw.sessionMgr = sm
}

// SetSysconfigStore injects the system configuration store for hot-reloadable settings.
func (gw *Gateway) SetSysconfigStore(store sysconfig.Store) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.sysconfigStore = store
	if gw.rateLimiter != nil {
		gw.rateLimiter.SetSysconfigStore(store)
	}
	if gw.sessionMgr != nil {
		gw.sessionMgr.SetSysconfigStore(store)
	}
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
				pkgmiddleware.SignInternalRequest(req, "gateway", gw.internalSecret)
			}
			if userID, ok := middleware.UserIDFromRequest(req); ok {
				req.Header.Set("X-User-ID", userID.String())
			}
			// Forward JWT scopes so backend services can check admin authorization.
			jwtClaims := middleware.ExtractJWTClaims(req)
			if len(jwtClaims.Scopes) > 0 {
				req.Header.Set("X-Scopes", strings.Join(jwtClaims.Scopes, ","))
				for _, s := range jwtClaims.Scopes {
				sl := strings.ToLower(s)
				if sl == "admin" || sl == "superadmin" || sl == "platform:admin" || sl == "tenant:admin" || sl == "administrator" || sl == "platform administrator" || sl == "tenant administrator" {
					req.Header.Set("X-User-Role", s)
					req.Header.Set("X-Is-Admin", "true")
					break
				}
			}
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
	// --- Scope-based route guard ---
	// After JWT validation, check if user has required scope for this path.
	if !gw.checkRouteScope(w, r) {
		return // 403 already written
	}

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

	// --- Deep health check (aggregated from all backends with latency) ---
	if r.URL.Path == "/healthz/deep" {
		if gw.healthChecker != nil {
			gw.healthChecker.DeepHandler().ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	// JWKS endpoint with caching headers (KB-295)
	if r.URL.Path == "/.well-known/jwks.json" {
		// Set Cache-Control for downstream caching (1 hour, stale-while-revalidate 5 min).
		w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
		// ETag support for conditional requests.
		if etag := gw.jwks.ETag(); etag != "" {
			if match := r.Header.Get("If-None-Match"); match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", etag)
		}
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

	// Device code approval page (RFC 8628) — used by CLI and other device flows
	if r.URL.Path == "/device" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(hostedDeviceApproveHTML))
		return
	}

	// OpenAPI JSON spec
	if r.URL.Path == "/api-docs" {
		serveOpenAPISpec(w, r)
		return
	}

	// --- Admin API (requires admin scope) ---
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
		if !gw.hasAdminScope(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "admin scope required"})
			return
		}
		if r.URL.Path == "/api/v1/admin/routes" && r.Method == http.MethodGet {
			gw.handleAdminRoutes(w, r)
			return
		}
		if r.URL.Path == "/api/v1/admin/stats" && r.Method == http.MethodGet {
			gw.handleAdminStats(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/toggle") && r.Method == http.MethodPost {
			gw.handleAdminToggleRoute(w, r)
			return
		}
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
	// Dashboard aggregate stats
	if r.URL.Path == "/api/v1/dashboard/stats" && r.Method == http.MethodGet {
		gw.handleDashboardStats(w, r)
		return
	}
	// Health overview for frontend
	if r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/health/services" {
		if r.Method == http.MethodGet {
			gw.handleHealthOverview(w, r)
			return
		}
	}
	// KB-294: Integration convenience endpoints.
	if r.URL.Path == "/api/v1/system/health" && r.Method == http.MethodGet {
		gw.handleSystemHealth(w, r)
		return
	}
	if r.URL.Path == "/api/v1/system/quickstart" && r.Method == http.MethodPost {
		gw.handleQuickstart(w, r)
		return
	}
	if r.URL.Path == "/api/v1/system/status" && r.Method == http.MethodGet {
		gw.handleSystemStatus(w, r)
		return
	}
	if r.URL.Path == "/api/v1/system/bootstrap" && r.Method == http.MethodPost {
		gw.handleSystemBootstrap(w, r)
		return
	}
	if r.URL.Path == "/api/v1/webhooks/events/catalog" && r.Method == http.MethodGet {
		gw.handleWebhookCatalog(w, r)
		return
	}
	// Tenant list/create handled by identity service via proxy route /api/v1/tenants
	if strings.HasPrefix(r.URL.Path, "/api/v1/tenants/") && !strings.HasPrefix(r.URL.Path, "/api/v1/tenants/resolve") && (r.Method == http.MethodGet || r.Method == http.MethodDelete) {
		if proxy, _ := gw.matchBackend("/api/v1/identity/tenants/"); proxy != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = r.URL.Path
			proxy.ServeHTTP(w, r2)
			return
		}
		gw.handleTenantDetail(w, r)
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
	// 5-dimensional rate limit endpoints.
	if r.URL.Path == "/api/v1/gateway/rate-limits" && r.Method == http.MethodGet {
		gw.handleGetRateLimits(w, r)
		return
	}
	if r.URL.Path == "/api/v1/gateway/rate-limits/status" && r.Method == http.MethodGet {
		gw.handleRateLimitStatus(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/gateway/rate-limits/") && r.Method == http.MethodPut {
		gw.handleUpdateRateLimit(w, r)
		return
	}
	// Device posture endpoints.
	if r.URL.Path == "/api/v1/devices/posture/evaluate" && r.Method == http.MethodPost {
		gw.handlePostureEvaluate(w, r)
		return
	}
	if r.URL.Path == "/api/v1/devices/posture/policies" && r.Method == http.MethodGet {
		gw.handlePostureGetPolicy(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/devices/posture/policies/") && r.Method == http.MethodPut {
		gw.handlePostureUpdatePolicy(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/devices/posture/") && r.Method == http.MethodGet {
		gw.handlePostureGet(w, r)
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

	// OpenAPI spec + Swagger UI.
	if r.URL.Path == "/swagger.json" && r.Method == http.MethodGet {
		middleware.OpenAPISpecHandler().ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/docs" && r.Method == http.MethodGet {
		middleware.SwaggerUIHandler().ServeHTTP(w, r)
		return
	}

	// Provisioning requests are proxied to the ggid-operator API server
	// (registered in cfg.Routes as /api/v1/provisioning). No stub fallback:
	// if the operator is unreachable the proxy returns 502, which is more
	// honest than fake empty data.

	// --- ZTNA Access Broker: /app/{slug}/* dynamic routing ---
	if gw.appRouter != nil {
		if gw.appRouter.HandleRequest(w, r) {
			return
		}
	}

	// Find matching backend by longest prefix
	backend, prefix := gw.matchBackend(r.URL.Path)
	if backend == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no route for this path"})
		return
	}

	// Path rewriting: map frontend API paths to backend service paths
	// Frontend calls /api/v1/mfa/status but auth service expects /api/v1/auth/mfa/status
	if prefix == "/api/v1/mfa" || prefix == "/api/v1/tokens" || prefix == "/api/v1/login-security" ||
		prefix == "/api/v1/password-history" || prefix == "/api/v1/delegation" ||
		prefix == "/api/v1/account-linking" || prefix == "/api/v1/consent" ||
		prefix == "/api/v1/notifications" || prefix == "/api/v1/introspection" ||
		prefix == "/api/v1/device-bindings" ||
		prefix == "/api/v1/api-keys" || prefix == "/api/v1/access-keys" ||
		prefix == "/api/v1/sessions" {
		// Rewrite /api/v1/<feature>/... -> /api/v1/auth/<feature>/...
		r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/"+strings.TrimPrefix(prefix, "/api/v1/")+"/", "/api/v1/auth/"+strings.TrimPrefix(prefix, "/api/v1/")+"/", 1)
		r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/"+strings.TrimPrefix(prefix, "/api/v1/"), "/api/v1/auth/"+strings.TrimPrefix(prefix, "/api/v1/"), 1)
	}

	// Identity rewrites: /api/v1/<feature>/... -> /api/v1/identity/<feature>/...
	if prefix == "/api/v1/dashboard" || prefix == "/api/v1/groups" || prefix == "/api/v1/flows" {
		r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/"+strings.TrimPrefix(prefix, "/api/v1/"), "/api/v1/identity/"+strings.TrimPrefix(prefix, "/api/v1/"), 1)
	}

	// Audit rewrites: /api/v1/<feature>/... -> /api/v1/audit/<feature>/...
	if prefix == "/api/v1/access-reviews" || prefix == "/api/v1/activity" || prefix == "/api/v1/exports" {
		r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/"+strings.TrimPrefix(prefix, "/api/v1/"), "/api/v1/audit/"+strings.TrimPrefix(prefix, "/api/v1/"), 1)
	}

	// Apply per-route timeout if configured
	if to, ok := gw.timeouts[prefix]; ok && to > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), to)
		defer cancel()
		r = r.WithContext(ctx)
	}
	// Wrap with circuit breaker: fail-fast 503 if backend is down
	h := middleware.CircuitMiddleware(prefix, gw.circuitRegistry, backend)
	h.ServeHTTP(w, r)
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
		if r.URL.Path == "/healthz" || r.URL.Path == "/healthz/live" || r.URL.Path == "/healthz/ready" || r.URL.Path == "/healthz/deep" || r.URL.Path == "/.well-known/jwks.json" || r.URL.Path == "/metrics" {
			isPublic = true
		}

		if isPublic {
			// Public path: no JWT required, but still validate if token present
			jwtMW := middleware.JWTAuth(gw.jwks, false, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			h := middleware.RequireAdminScope(gw) // RBAC: block non-admin from management endpoints
			h = middleware.CheckConsent(gw.cfg.DatabaseURL)(h) // Consent: block platform admin from tenant data without consent
			// CAE: check jti blocklist AFTER JWTAuth (needs jti in context)
			if gw.caeCheck != nil {
				h = gw.caeCheck(h)
			}
			h = jwtMW(h)
			if gw.sessionMgr != nil {
				h = gw.sessionMgr.SessionTimeoutMiddleware(middleware.DefaultSessionTimeoutConfig())(h)
			}
			h.ServeHTTP(w, r)
		} else {
			// Protected path: JWT required
			jwtMW := middleware.JWTAuth(gw.jwks, true, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
			h := middleware.RequireAdminScope(gw) // RBAC: block non-admin from management endpoints
			h = middleware.CheckConsent(gw.cfg.DatabaseURL)(h) // Consent: block platform admin from tenant data without consent
			// CAE: check jti blocklist AFTER JWTAuth (needs jti in context)
			if gw.caeCheck != nil {
				h = gw.caeCheck(h)
			}
			h = jwtMW(h)
			if gw.sessionMgr != nil {
				h = gw.sessionMgr.SessionTimeoutMiddleware(middleware.DefaultSessionTimeoutConfig())(h)
			}
			h.ServeHTTP(w, r)
		}
	})

	// Apply outer middleware: PanicRecovery → SecurityHeaders → CORS → RequestID → StructuredLogging → RateLimit → BotDetect → TenantResolver → Timeout → MaxBodySize → inner
	logger := middleware.NewStructuredLogger("ggid-gateway")
	handler := middleware.MaxBodySize(gw.maxBodySize())(inner)
	handler = middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig())(handler)
	handler = middleware.TenantResolver(gw.cfg.DomainSuffix)(handler)
	handler = middleware.BotDetect(handler)
	handler = gw.rateLimiter.Middleware(handler)
	handler = middleware.ContentTypeValidator(handler)
	handler = middleware.RequestLogger(logger)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Gzip(handler)
	handler = middleware.CORS(handler)
	handler = middleware.HostValidation(gw.hostValidationConfig())(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.PanicRecovery(logger)(handler)

	return handler
}

// maxBodySize returns the configured maximum request body size.

// adminOnlyPaths are paths that require tenant:admin or platform:admin scope.
var adminOnlyPaths = []string{
	"/api/v1/users", "/api/v1/roles", "/api/v1/audit/events",
	"/api/v1/policies", "/api/v1/webhooks", "/api/v1/oauth/clients",
	"/api/v1/settings/", "/api/v1/admin/", "/api/v1/identity/dashboard",
	"/api/v1/tenants",
}

// platformOnlyPaths require platform:admin scope.
var platformOnlyPaths = []string{
	"/api/v1/system/", "/api/v1/tenants/create",
	"/api/v1/org/tenants/suspend", "/api/v1/org/tenants/activate",
	"/api/v1/admin/audit/global", "/api/v1/admin/threats/dashboard",
}

// checkRouteScope verifies the user has the required scope for the request path.
// Returns true if access is allowed, false if 403 was written.
func (gw *Gateway) checkRouteScope(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	// Get scopes from JWT claims
	claims := middleware.ExtractJWTClaims(r)
	if len(claims.Scopes) == 0 {
		return true // no JWT — let auth middleware handle
	}

	hasPlatform := false
	hasTenant := false
	for _, sc := range claims.Scopes {
		scl := strings.ToLower(sc)
		if scl == "platform:admin" || scl == "admin" || scl == "superadmin" ||
			scl == "platform administrator" || scl == "administrator" {
			hasPlatform = true
			hasTenant = true
		}
		if scl == "tenant:admin" || scl == "manager" || scl == "tenant administrator" {
			hasTenant = true
		}
	}

	// Check platform-only paths
	for _, prefix := range platformOnlyPaths {
		if strings.HasPrefix(path, prefix) && !hasPlatform {
			writeGatewayJSONError(w, http.StatusForbidden, "platform admin access required")
			return false
		}
	}

		// Check tenant-admin paths (but allow self-service endpoints)
	for _, prefix := range adminOnlyPaths {
		if strings.HasPrefix(path, prefix) && !hasTenant {
			// Allow /api/v1/users/me for self-service
			if path == "/api/v1/users/me" || strings.HasPrefix(path, "/api/v1/users/me/") {
				continue
			}
			// Allow /api/v1/tenants/resolve (public lookup)
			if strings.HasPrefix(path, "/api/v1/tenants/resolve") {
				continue
			}
			writeGatewayJSONError(w, http.StatusForbidden, "admin access required")
			return false
		}
	}

	return true
}

// maxBodySize returns the configured maximum request body size.
// Defaults to 10 MiB; override via GATEWAY_MAX_BODY_SIZE_BYTES env var.
func (gw *Gateway) maxBodySize() int64 {
	const defaultMax = 10 * 1024 * 1024
	if gw.cfg != nil && gw.cfg.MaxBodySize > 0 {
		return gw.cfg.MaxBodySize
	}
	if v := os.Getenv("GATEWAY_MAX_BODY_SIZE_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultMax
}

// hostValidationConfig returns the Host header validation config.
// Allowed hosts can be set via GATEWAY_ALLOWED_HOSTS (comma-separated).
// Empty allowlist disables validation (allow all), which is the safe default for development.
func (gw *Gateway) hostValidationConfig() middleware.HostValidationConfig {
	cfg := middleware.HostValidationConfig{}
	if gw.cfg != nil && len(gw.cfg.AllowedHosts) > 0 {
		cfg.AllowedHosts = gw.cfg.AllowedHosts
		return cfg
	}
	if v := os.Getenv("GATEWAY_ALLOWED_HOSTS"); v != "" {
		cfg.AllowedHosts = strings.Split(v, ",")
		for i := range cfg.AllowedHosts {
			cfg.AllowedHosts[i] = strings.TrimSpace(cfg.AllowedHosts[i])
		}
	}
	return cfg
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
// hasAdminScope checks if the request has admin scope in JWT claims.
func (gw *Gateway) hasAdminScope(r *http.Request) bool {
	claims := middleware.ExtractJWTClaims(r)
	for _, s := range claims.Scopes {
		if s == "admin" || s == "ggid:admin" {
			return true
		}
	}
	return false
}

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
				pkgmiddleware.SignInternalRequest(req, "gateway", gw.internalSecret)
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

// serveOpenAPISpec writes the OpenAPI 3.1 JSON spec (dynamically generated
// from route scanning + schema definitions).
func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	spec := middleware.GenerateOpenAPISpec()
	_ = json.NewEncoder(w).Encode(spec)
}

// handleDashboardStats returns aggregate dashboard statistics.
// Proxies to identity service for real data.
func (gw *Gateway) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	// Use identity proxy for real stats
	if proxy, prefix := gw.matchBackend("/api/v1/identity/dashboard/stats"); proxy != nil {
		_ = prefix
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/api/v1/identity/dashboard/stats"
		proxy.ServeHTTP(w, r2)
		return
	}
	// Fallback
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total_users":         0,
		"active_sessions":     0,
		"login_rate_per_hour": 0,
		"mfa_adoption_pct":    0,
	})
}

// handleHealthOverview returns service health for the frontend.
func (gw *Gateway) handleHealthOverview(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	services := []map[string]interface{}{}
	// Add gateway itself first
	services = append(services, map[string]interface{}{
		"name":   "gateway",
		"url":    "self",
		"status": "healthy",
	})
	if gw.healthChecker != nil {
		// Use real health checker results
		status := gw.healthChecker.CheckAll(context.Background())
		for _, svc := range status.Services {
			services = append(services, map[string]interface{}{
				"name":   svc.Name,
				"status": svc.Status,
			})
		}
	} else {
		// Fallback: list routes as healthy
		for prefix, backendURL := range gw.cfg.Routes {
			name := strings.TrimPrefix(prefix, "/api/v1/")
			if name == "" {
				name = strings.TrimPrefix(prefix, "/")
			}
			services = append(services, map[string]interface{}{
				"name":   name,
				"url":    backendURL,
				"status": "healthy",
			})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"services": services,
	})
}

// --- 5-Dimensional Rate Limit Endpoints ---

// handleGetRateLimits returns all configured rate limit tiers.
func (gw *Gateway) handleGetRateLimits(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.multiDimLimiter == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "multi-dimensional rate limiter not configured"})
		return
	}
	_ = json.NewEncoder(w).Encode(gw.multiDimLimiter.AllTiers())
}

// handleRateLimitStatus returns current rate limit usage for the caller.
func (gw *Gateway) handleRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.multiDimLimiter == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "multi-dimensional rate limiter not configured"})
		return
	}
	tier, tenantID, userID, apiKey, ip, endpoint := middleware.DefaultTierResolver(r)
	usage := gw.multiDimLimiter.GetUsage(middleware.Tier(tier), tenantID, userID, apiKey, ip, endpoint)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tier":    tier,
		"usage":   usage,
	})
}

// handleUpdateRateLimit updates limits for a specific tier.
func (gw *Gateway) handleUpdateRateLimit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.multiDimLimiter == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "multi-dimensional rate limiter not configured"})
		return
	}
	// Extract tier from path: /api/v1/gateway/rate-limits/:tier
	tierStr := strings.TrimPrefix(r.URL.Path, "/api/v1/gateway/rate-limits/")
	if tierStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tier required in path"})
		return
	}

	var req struct {
		BurstPerMin      map[string]int `json:"burst_per_min"`
		SustainedPerHour map[string]int `json:"sustained_per_hour"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	cfg := middleware.MultiDimTierConfig{
		Tenant:   middleware.MultiDimRateLimit{BurstPerMin: req.BurstPerMin["tenant"], SustainedPerHour: req.SustainedPerHour["tenant"]},
		User:     middleware.MultiDimRateLimit{BurstPerMin: req.BurstPerMin["user"], SustainedPerHour: req.SustainedPerHour["user"]},
		APIKey:   middleware.MultiDimRateLimit{BurstPerMin: req.BurstPerMin["api_key"], SustainedPerHour: req.SustainedPerHour["api_key"]},
		IP:       middleware.MultiDimRateLimit{BurstPerMin: req.BurstPerMin["ip"], SustainedPerHour: req.SustainedPerHour["ip"]},
		Endpoint: middleware.MultiDimRateLimit{BurstPerMin: req.BurstPerMin["endpoint"], SustainedPerHour: req.SustainedPerHour["endpoint"]},
	}
	gw.multiDimLimiter.UpdateTier(middleware.Tier(tierStr), cfg)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "updated",
		"tier":   tierStr,
		"config": cfg,
	})
}

// SetPostureEngine injects the device posture evaluation engine.
func (gw *Gateway) SetPostureEngine(engine *posture.Engine) {
	gw.postureEngine = engine
}

// SetPostureDropHandler injects a callback for posture-drop session revocation.
// When device posture drops below threshold, this callback triggers session invalidation
// via the auth service's internal revoke-user endpoint.
func (gw *Gateway) SetPostureDropHandler(fn func(ctx context.Context, tenantID, userID string, score int)) {
	gw.postureDropFn = fn
}

func (gw *Gateway) handlePostureEvaluate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.postureEngine == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "posture engine not configured"})
		return
	}
	var input posture.PostureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	result := gw.postureEngine.Evaluate(tenantID, input)
	gw.postureEngine.PersistResult(r.Context(), tenantID, result)

	// Posture drop: when device is non-compliant or blocked, revoke sessions.
	if !result.Compliant || result.Action == "block" {
		if gw.postureDropFn != nil {
			userID := r.Header.Get("X-User-ID")
			if userID != "" {
				gw.postureDropFn(r.Context(), tenantID, userID, result.Score)
			}
		}
	}

	_ = json.NewEncoder(w).Encode(result)
}

func (gw *Gateway) handlePostureGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.postureEngine == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "posture engine not configured"})
		return
	}
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/posture/")
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	result, _ := gw.postureEngine.GetLatestScore(r.Context(), tenantID, deviceID)
	if result == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no posture data for device"})
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (gw *Gateway) handlePostureGetPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.postureEngine == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "posture engine not configured"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	_ = json.NewEncoder(w).Encode(gw.postureEngine.GetPolicy(tenantID))
}

func (gw *Gateway) handlePostureUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if gw.postureEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "posture engine not configured"})
		return
	}
	tenantID := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/posture/policies/")
	if tenantID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}
	var policy posture.PosturePolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}
	gw.postureEngine.SetPolicy(tenantID, policy)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated", "tenant_id": tenantID})
}
