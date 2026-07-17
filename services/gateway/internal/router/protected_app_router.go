package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// ProtectedApp represents a ZTNA-protected application loaded into the gateway.
type ProtectedApp struct {
	ID            string                   `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	Name          string                   `json:"name"`
	Slug          string                   `json:"slug"`
	UpstreamURL   string                   `json:"upstream_url"`
	AuthMode      string                   `json:"auth_mode"`
	AccessPolicy  map[string]any           `json:"access_policy"`
	InjectHeaders []map[string]any         `json:"inject_headers"`
	HealthStatus  string                   `json:"health_status"`
	RateLimitPerMin int                    `json:"rate_limit_per_min"`
}

// AppAccessLogEntry records a single proxied request.
type AppAccessLogEntry struct {
	AppID          string
	TenantID       string
	UserID         string
	UserName       string
	Method         string
	Path           string
	StatusCode     int
	ResponseTimeMs int64
	IPAddress      string
	UserAgent      string
	PDPDecision    string
	PDPReason      string
}

// ProtectedAppRouter manages dynamic routes for ZTNA protected apps.
// Routes are matched by /app/{slug}/* prefix and proxied to upstream_url
// after PDP evaluation and header injection.
type ProtectedAppRouter struct {
	mu       sync.RWMutex
	apps     map[string]*ProtectedApp // slug → app
	proxies  map[string]*httputil.ReverseProxy // slug → reverse proxy
	auditPub middleware.AuditPublisher // NATS publisher for access logs
	postureCache sync.Map // tenantID:deviceID → postureCacheEntry
}

func NewProtectedAppRouter() *ProtectedAppRouter {
	return &ProtectedAppRouter{
		apps:    make(map[string]*ProtectedApp),
		proxies: make(map[string]*httputil.ReverseProxy),
	}
}

// SetAuditPublisher injects a NATS publisher for persisting access logs.
func (r *ProtectedAppRouter) SetAuditPublisher(pub middleware.AuditPublisher) {
	r.auditPub = pub
}

// RegisterApp adds or updates a protected app route.
func (r *ProtectedAppRouter) RegisterApp(app *ProtectedApp) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.apps[app.Slug] = app

	// Create reverse proxy for upstream.
	upstream, err := url.Parse(app.UpstreamURL)
	if err != nil {
		log.Printf("ZTNA: failed to parse upstream URL %s: %v", app.UpstreamURL, err)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Strip /app/{slug} prefix before forwarding.
		path := strings.TrimPrefix(req.URL.Path, "/app/"+app.Slug)
		if path == "" {
			path = "/"
		}
		req.URL.Path = path
		// Set Host to upstream host.
		req.Host = upstream.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("ZTNA proxy error for app %s: %v", app.Slug, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "upstream unavailable"})
	}

	r.proxies[app.Slug] = proxy
	log.Printf("ZTNA: registered app %s → %s", app.Slug, app.UpstreamURL)
}

// UnregisterApp removes a protected app route.
func (r *ProtectedAppRouter) UnregisterApp(slug string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.apps, slug)
	delete(r.proxies, slug)
	log.Printf("ZTNA: unregistered app %s", slug)
}

// GetApp returns the protected app for a slug.
func (r *ProtectedAppRouter) GetApp(slug string) (*ProtectedApp, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[slug]
	return app, ok
}

// ListApps returns all registered apps.
func (r *ProtectedAppRouter) ListApps() []*ProtectedApp {
	r.mu.RLock()
	defer r.mu.RUnlock()
	apps := make([]*ProtectedApp, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	return apps
}

// HandleRequest processes a /app/{slug}/* request:
// 1. Extract slug from path
// 2. Look up protected app
// 3. Evaluate access policy (PDP)
// 4. Inject headers (clear forged + inject GGID identity)
// 5. Proxy to upstream
// Returns true if handled, false if not an /app/ path.
func (r *ProtectedAppRouter) HandleRequest(w http.ResponseWriter, req *http.Request) bool {
	if !strings.HasPrefix(req.URL.Path, "/app/") {
		return false
	}

	// Extract slug from /app/{slug}/...
	pathParts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/app/"), "/", 2)
	slug := pathParts[0]
	if slug == "" {
		return false
	}

	r.mu.RLock()
	app, appOK := r.apps[slug]
	proxy := r.proxies[slug]
	r.mu.RUnlock()

	if !appOK || proxy == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "application not found"})
		return true
	}

	start := time.Now()

	// --- PDP: Evaluate access policy ---
	decision := r.evaluatePolicy(app, req)
	if decision.Decision == "deny" {
		r.logAccess(app, req, http.StatusForbidden, time.Since(start), "deny", decision.Reason)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  "access denied",
			"reason": decision.Reason,
		})
		return true
	}
	if decision.Decision == "stepup" {
		r.logAccess(app, req, http.StatusPaymentRequired, time.Since(start), "stepup", decision.Reason)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Require-MFA", "true")
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  "additional authentication required",
			"reason": decision.Reason,
		})
		return true
	}

	// --- Header injection: clear forged headers + inject identity ---
	r.injectHeaders(app, req)

	// --- Proxy to upstream ---
	proxy.ServeHTTP(w, req)

	r.logAccess(app, req, http.StatusOK, time.Since(start), "allow", "")
	return true
}

// evaluatePolicy performs simplified ABAC evaluation for the app.
// When $security.device_trusted or $security.compliance_score conditions
// are present, queries the identity service device posture API (cached 5min).
func (r *ProtectedAppRouter) evaluatePolicy(app *ProtectedApp, req *http.Request) PDPResult {
	if len(app.AccessPolicy) == 0 {
		return PDPResult{Decision: "allow"}
	}

	conditions, ok := app.AccessPolicy["conditions"].(map[string]any)
	if !ok {
		return PDPResult{Decision: "allow"}
	}

	andConds, ok := conditions["and"].([]any)
	if !ok {
		return PDPResult{Decision: "allow"}
	}

	// Extract user info from JWT claims (injected by gateway JWT middleware).
	userID := req.Header.Get("X-User-ID")
	userRole := req.Header.Get("X-User-Role")

	// Resolve device posture from identity service (cached 5min).
	// Device ID comes from the JWT "device_id" claim or X-Device-ID header.
	deviceID := req.Header.Get("X-Device-ID")
	tenantID := req.Header.Get("X-Tenant-ID")
	deviceTrusted := req.Header.Get("X-Device-Trusted") == "true"
	complianceScore := 0

	// If device ID present, query posture API for real-time trust.
	if deviceID != "" && tenantID != "" {
		posture := r.resolveDevicePosture(tenantID, deviceID)
		if posture != nil {
			deviceTrusted = posture.Compliant
			complianceScore = posture.PostureScore
		}
	}

	for _, cond := range andConds {
		condMap, ok := cond.(map[string]any)
		if !ok {
			continue
		}
		for key, expected := range condMap {
			switch key {
			case "$user.role":
				if fmt.Sprintf("%v", expected) != userRole {
					return PDPResult{Decision: "deny", Reason: "role mismatch: required " + fmt.Sprintf("%v", expected)}
				}
			case "$security.device_trusted":
				if expected == true && !deviceTrusted {
					return PDPResult{Decision: "stepup", Reason: "trusted device required"}
				}
			case "$user.id":
				if fmt.Sprintf("%v", expected) != userID {
					return PDPResult{Decision: "deny", Reason: "user mismatch"}
				}
			case "$security.compliance_score":
				// Support $gte operator: {"$security.compliance_score": {"$gte": 70}}
				if expMap, ok := expected.(map[string]any); ok {
					if gte, has := expMap["$gte"]; has {
						if minScore, ok := toInt(gte); ok && complianceScore < minScore {
							return PDPResult{Decision: "stepup", Reason: fmt.Sprintf("device compliance score %d < required %d", complianceScore, minScore)}
						}
					}
				} else if minScore, ok := toInt(expected); ok {
					if complianceScore < minScore {
						return PDPResult{Decision: "stepup", Reason: fmt.Sprintf("device compliance score %d < required %d", complianceScore, minScore)}
					}
				}
			}
		}
	}

	return PDPResult{Decision: "allow"}
}

// DevicePosture is the cached posture response from identity service.
type DevicePosture struct {
	Compliant    bool `json:"compliant"`
	PostureScore int  `json:"posture_score"`
}

// resolveDevicePosture queries identity service with 5min cache.
func (r *ProtectedAppRouter) resolveDevicePosture(tenantID, deviceID string) *DevicePosture {
	cacheKey := tenantID + ":" + deviceID

	// Check cache.
	if cached, ok := r.postureCache.Load(cacheKey); ok {
		if entry, ok := cached.(postureCacheEntry); ok {
			if time.Since(entry.fetchedAt) < 5*time.Minute {
				return entry.posture
			}
		}
	}

	// Query identity service.
	url := "http://localhost:8081/api/v1/identity/devices/" + deviceID + "/posture"
	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("X-Tenant-ID", tenantID)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()

	var dp DevicePosture
	if json.NewDecoder(resp.Body).Decode(&dp) != nil {
		return nil
	}

	// Cache result.
	r.postureCache.Store(cacheKey, postureCacheEntry{posture: &dp, fetchedAt: time.Now()})
	return &dp
}

type postureCacheEntry struct {
	posture   *DevicePosture
	fetchedAt time.Time
}

// toInt safely converts any to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

// PDPResult is the policy decision for a ZTNA request.
type PDPResult struct {
	Decision string // allow, deny, stepup
	Reason   string
}

// injectHeaders clears forged inbound headers and injects GGID identity headers.
func (r *ProtectedAppRouter) injectHeaders(app *ProtectedApp, req *http.Request) {
	// Clear any inbound forged identity headers.
	forgedHeaders := []string{
		"X-GGID-User", "X-GGID-Roles", "X-GGID-Tenant",
		"X-WebAuth-User", "X-WebAuth-Roles", "X-Forwarded-User",
	}
	for _, h := range forgedHeaders {
		req.Header.Del(h)
	}

	// Extract identity from gateway JWT middleware headers.
	userID := req.Header.Get("X-User-ID")
	userEmail := req.Header.Get("X-User-Email")
	userRoles := req.Header.Get("X-User-Roles")
	tenantID := req.Header.Get("X-Tenant-ID")

	// Inject GGID identity headers (trusted by upstream apps).
	if userID != "" {
		req.Header.Set("X-GGID-User", userID)
	}
	if userEmail != "" {
		req.Header.Set("X-WebAuth-User", userEmail)
	}
	if userRoles != "" {
		req.Header.Set("X-GGID-Roles", userRoles)
		req.Header.Set("X-WebAuth-Roles", userRoles)
	}
	if tenantID != "" {
		req.Header.Set("X-GGID-Tenant", tenantID)
	}

	// Apply app-specific custom header injection config.
	for _, hdr := range app.InjectHeaders {
		name, _ := hdr["name"].(string)
		value, _ := hdr["value"].(string)
		if name == "" {
			continue
		}
		// Template substitution: $user.email → actual email.
		value = strings.ReplaceAll(value, "$user.email", userEmail)
		value = strings.ReplaceAll(value, "$user.id", userID)
		value = strings.ReplaceAll(value, "$user.roles_csv", userRoles)
		req.Header.Set(name, value)
	}
}

// logAccess records an access log entry via NATS (consumed by identity service → app_access_logs table).
func (r *ProtectedAppRouter) logAccess(app *ProtectedApp, req *http.Request, statusCode int, duration time.Duration, decision, reason string) {
	userID := req.Header.Get("X-User-ID")

	// Log to stdout (always).
	log.Printf("ZTNA access: app=%s user=%s %s %s → %d (%s, %dms)",
		app.Slug, userID, req.Method, req.URL.Path, statusCode, decision, duration.Milliseconds())

	// Publish to NATS for DB persistence (app_access_logs table).
	if r.auditPub != nil {
		event := &middleware.AuditEvent{
			Timestamp:  time.Now(),
			Method:     req.Method,
			Path:       req.URL.Path,
			StatusCode: statusCode,
			LatencyMs:  float64(duration.Milliseconds()),
			TenantID:   app.TenantID,
			UserID:     userID,
			ClientIP:   req.RemoteAddr,
			UserAgent:  req.UserAgent(),
			RequestID:  req.Header.Get("X-Request-ID"),
		}
		_ = r.auditPub.Publish(event)
	}
}
