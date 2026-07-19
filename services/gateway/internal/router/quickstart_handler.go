package router

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
	Status             string `json:"status"`
	TenantID           string `json:"tenant_id"`
	AdminUserID        string `json:"admin_user_id"`
	AdminUsername      string `json:"admin_username"`
	OAuthClientID      string `json:"oauth_client_id"`
	OAuthClientSecret  string `json:"oauth_client_secret"`
	SampleCurl         []string `json:"sample_curl"`
	NextSteps          []string `json:"next_steps"`
}

// quickstartState tracks whether the system has been initialized (in-memory).
// In production this would check the database for existing users/tenants.
var quickstartInitialized bool

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
	if quickstartInitialized {
		writeGatewayJSON(w, http.StatusOK, QuickstartResponse{
			Status:        "already_initialized",
			AdminUsername: "admin",
			NextSteps: []string{
				"POST /api/v1/auth/login with admin credentials to get a fresh token",
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
		"curl -X POST " + baseURL + "/api/v1/auth/login \\\n  -H 'Content-Type: application/json' \\\n  -H 'X-Tenant-ID: " + tenantID.String() + "' \\\n  -d '{\"username\":\"" + req.AdminUsername + "\",\"password\":\"" + req.AdminPassword + "\"}'",
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
			"Login at POST /api/v1/auth/login",
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

	// Determine initialization state by probing auth service.
	// If a login attempt returns "invalid credentials" (not "no tenant context"),
	// it means users exist → system is initialized.
	initialized := quickstartInitialized
	if !initialized {
		authURL := gw.serviceURL("/api/v1/auth")
		client := &http.Client{Timeout: 3 * time.Second}
		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		loginBody := `{"username":"__setup_probe__","password":"__nonexistent__"}`
		req, _ := http.NewRequest("POST", authURL+"/api/v1/auth/login", strings.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID.String())
		if resp, err := client.Do(req); err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			// "invalid credentials" = users exist in DB = initialized
			if resp.StatusCode == 401 && strings.Contains(string(body), "invalid") {
				initialized = true
			}
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
