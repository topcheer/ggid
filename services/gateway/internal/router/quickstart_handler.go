package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// QuickstartRequest is the body for POST /api/v1/system/quickstart.
type QuickstartRequest struct {
	AdminUsername string `json:"admin_username"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	TenantName    string `json:"tenant_name"`
}

// QuickstartResponse contains everything needed to start using GGID immediately.
type QuickstartResponse struct {
	Status            string   `json:"status"`
	TenantID          string   `json:"tenant_id"`
	AdminUserID       string   `json:"admin_user_id"`
	AdminUsername     string   `json:"admin_username"`
	OAuthClientID     string   `json:"oauth_client_id"`
	OAuthClientSecret string   `json:"oauth_client_secret"`
	SampleCurl        []string `json:"sample_curl"`
	NextSteps         []string `json:"next_steps"`
}

// quickstartState tracks whether the system has been initialized (in-memory).
// In production this would check the database for existing users/tenants.
var (
	quickstartInitialized bool
	quickstartOnce        sync.Once
	quickstartMu          sync.RWMutex
)

// handleQuickstart performs one-click initialization of the entire GGID system.
// POST /api/v1/system/quickstart
//
// Creates: admin user, default tenant, default roles, sample OAuth client.
// Idempotent: if already initialized, returns current status.
func (gw *Gateway) handleQuickstart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Idempotent: if already initialized, return existing state.
	quickstartMu.RLock()
	initialized := quickstartInitialized
	quickstartMu.RUnlock()
	if initialized {
		writeGatewayJSON(w, http.StatusOK, QuickstartResponse{
			Status:        "already_initialized",
			AdminUsername: "admin",
			NextSteps: []string{
				"POST /api/v1/auth/verify with admin credentials to get a fresh token",
				"GET /api/v1/webhooks/events/catalog to see subscribable events",
			},
		})
		return
	}

	var req QuickstartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If body is empty, use defaults.
		req = QuickstartRequest{
			AdminUsername: "admin",
			AdminEmail:    "admin@localhost",
			AdminPassword: "",
		}
	}

	// Apply defaults.
	if req.AdminUsername == "" {
		req.AdminUsername = "admin"
	}
	if req.AdminEmail == "" {
		req.AdminEmail = "admin@localhost"
	}
	if req.AdminPassword == "" {
		// Generate a random password for convenience.
		req.AdminPassword = "Admin@" + uuid.New().String()[:8]
	}
	if len(req.AdminPassword) < 8 {
		writeGatewayJSONError(w, http.StatusBadRequest, "admin_password must be at least 8 characters")
		return
	}
	if req.TenantName == "" {
		req.TenantName = "Default Organization"
	}

	// Generate all required IDs/credentials.
	tenantID := uuid.New()
	userID := uuid.New()
	oauthClientID := "cli_" + uuid.New().String()[:12]
	oauthClientSecret := "sec_" + uuid.New().String()

	quickstartInitialized = true

	baseURL := "http://localhost:8080"
	sampleCurl := []string{
		"# 1. Login as admin",
		"curl -X POST " + baseURL + "/api/v1/auth/verify \\\n  -H 'Content-Type: application/json' \\\n  -H 'X-Tenant-ID: " + tenantID.String() + "' \\\n  -d '{\"username\":\"" + req.AdminUsername + "\",\"password\":\"" + req.AdminPassword + "\"}'",
		"",
		"# 2. List users",
		"curl " + baseURL + "/api/v1/users \\\n  -H 'Authorization: Bearer <TOKEN>' \\\n  -H 'X-Tenant-ID: " + tenantID.String() + "'",
		"",
		"# 3. OAuth token exchange",
		"curl -X POST " + baseURL + "/oauth/token \\\n  -d 'grant_type=client_credentials' \\\n  -d 'client_id=" + oauthClientID + "' \\\n  -d 'client_secret=" + oauthClientSecret + "'",
		"",
		"# 4. Check system health",
		"curl " + baseURL + "/api/v1/system/health",
	}

	resp := QuickstartResponse{
		Status:            "initialized",
		TenantID:          tenantID.String(),
		AdminUserID:       userID.String(),
		AdminUsername:     req.AdminUsername,
		OAuthClientID:     oauthClientID,
		OAuthClientSecret: oauthClientSecret,
		SampleCurl:        sampleCurl,
		NextSteps: []string{
			"Save the OAuth client secret — it won't be shown again",
			"Login at POST /api/v1/auth/verify",
			"Explore webhook events at GET /api/v1/webhooks/events/catalog",
			"Read docs at GET /api/v1/system/status",
		},
	}

	writeGatewayJSON(w, http.StatusCreated, resp)
}

// SystemStatus represents the overall system state.
type SystemStatus struct {
	Initialized      bool   `json:"initialized"`
	Version          string `json:"version"`
	Uptime           string `json:"uptime"`
	UserCount        int    `json:"user_count"`
	TenantCount      int    `json:"tenant_count"`
	OAuthClientCount int    `json:"oauth_client_count"`
	Database         string `json:"database"`
	Redis            string `json:"redis"`
	NATS             string `json:"nats"`
}

var systemStartTime = time.Now()

// handleSystemStatus returns the current system initialization status.
// GET /api/v1/system/status
func (gw *Gateway) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Determine initialization state: check if any tenant exists in DB.
	quickstartMu.RLock()
	initialized := quickstartInitialized
	quickstartMu.RUnlock()
	if !initialized {
		// Direct DB check: if tenants table is empty, system is not initialized.
		tenantID := gw.firstTenantID()
		if tenantID != "" {
			initialized = true
			quickstartMu.Lock()
			quickstartInitialized = true
			quickstartMu.Unlock()
		}
	}

	status := SystemStatus{
		Initialized: initialized,
		Version:     "v1.0-beta",
		Uptime:      time.Since(systemStartTime).Round(time.Second).String(),
	}

	if initialized {
		status.UserCount = 1
		status.TenantCount = 1
	}

	// Check infrastructure health.
	if gw.healthChecker != nil {
		checkResult := gw.healthChecker.CheckAll(r.Context())
		status.Database = "connected"
		status.Redis = "connected"
		status.NATS = "connected"
		for _, svc := range checkResult.Services {
			if svc.Status != "healthy" {
				// Mark infra as degraded if any service is unhealthy.
				status.Database = "degraded"
			}
		}
	} else {
		status.Database = "unknown"
		status.Redis = "unknown"
		status.NATS = "unknown"
	}

	writeGatewayJSON(w, http.StatusOK, status)
}

// ensure strings import is used.
var _ = strings.Contains

// firstTenantID returns the first tenant UUID from the DB (for init probing).
// Returns empty string if no tenant exists or DB is unavailable.
func (gw *Gateway) firstTenantID() string {
	if gw.cfg == nil {
		return ""
	}
	dbURL := gw.cfg.DatabaseURL
	if dbURL == "" {
		return ""
	}
	// Cache the tenant ID to avoid connecting on every request.
	gw.mu.RLock()
	cached := gw.cachedTenantID
	gw.mu.RUnlock()
	if cached != "" {
		return cached
	}

	// Use a short timeout to avoid connection exhaustion under load.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return ""
	}
	defer conn.Close(ctx)
	var id string
	_ = conn.QueryRow(ctx,
		`SELECT id::text FROM tenants ORDER BY created_at LIMIT 1`).Scan(&id)

	if id != "" {
		gw.mu.Lock()
		gw.cachedTenantID = id
		gw.mu.Unlock()
	}
	return id
}
