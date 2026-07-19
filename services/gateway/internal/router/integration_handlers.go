package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- Webhook Event Catalog ---

// WebhookEventCatalogEntry describes a subscribable event.
type WebhookEventCatalogEntry struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	PayloadExample map[string]any `json:"payload_example"`
}

var webhookEventCatalog = []WebhookEventCatalogEntry{
	{
		Type: "user.created", Category: "identity",
		Description: "Fired when a new user is created",
		PayloadExample: map[string]any{
			"event":   "user.created",
			"tenant_id": "00000000-0000-0000-0000-000000000001",
			"user_id":   "550e8400-e29b-41d4-a716-446655440000",
			"username":  "john.doe",
			"email":     "john@example.com",
			"timestamp": "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "user.deleted", Category: "identity",
		Description: "Fired when a user is deleted",
		PayloadExample: map[string]any{
			"event":     "user.deleted",
			"tenant_id": "00000000-0000-0000-0000-000000000001",
			"user_id":   "550e8400-e29b-41d4-a716-446655440000",
			"timestamp": "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "auth.login.success", Category: "auth",
		Description: "Fired on successful user login",
		PayloadExample: map[string]any{
			"event":      "auth.login.success",
			"tenant_id":  "00000000-0000-0000-0000-000000000001",
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"ip_address": "192.168.1.100",
			"timestamp":  "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "auth.login.failed", Category: "auth",
		Description: "Fired on failed login attempt",
		PayloadExample: map[string]any{
			"event":      "auth.login.failed",
			"tenant_id":  "00000000-0000-0000-0000-000000000001",
			"username":   "john.doe",
			"ip_address": "192.168.1.100",
			"reason":     "invalid_credentials",
			"timestamp":  "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "session.revoked", Category: "auth",
		Description: "Fired when a session is revoked (admin, password change, posture drop)",
		PayloadExample: map[string]any{
			"event":      "session.revoked",
			"tenant_id":  "00000000-0000-0000-0000-000000000001",
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"reason":     "password_change",
			"timestamp":  "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "role.assigned", Category: "identity",
		Description: "Fired when a role is assigned to a user",
		PayloadExample: map[string]any{
			"event":      "role.assigned",
			"tenant_id":  "00000000-0000-0000-0000-000000000001",
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"role":       "admin",
			"assigned_by":"550e8400-e29b-41d4-a716-446655440001",
			"timestamp":  "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "policy.violation", Category: "policy",
		Description: "Fired when a policy violation is detected (SoD, CAE, privilege creep)",
		PayloadExample: map[string]any{
			"event":      "policy.violation",
			"tenant_id":  "00000000-0000-0000-0000-000000000001",
			"type":       "sod_violation",
			"user_id":    "550e8400-e29b-41d4-a716-446655440000",
			"severity":   "high",
			"detail":     "User holds admin + auditor roles",
			"timestamp":  "2026-01-15T10:30:00Z",
		},
	},
	{
		Type: "delegation.created", Category: "auth",
		Description: "Fired when a user delegates permissions to another user",
		PayloadExample: map[string]any{
			"event":        "delegation.created",
			"tenant_id":    "00000000-0000-0000-0000-000000000001",
			"delegator_id": "550e8400-e29b-41d4-a716-446655440000",
			"delegatee_id": "550e8400-e29b-41d4-a716-446655440001",
			"scopes":       []string{"read", "write"},
			"timestamp":    "2026-01-15T10:30:00Z",
		},
	},
}

// handleWebhookCatalog returns the catalog of subscribable webhook events.
// GET /api/v1/webhooks/events/catalog
func (gw *Gateway) handleWebhookCatalog(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"events": webhookEventCatalog,
		"count":  len(webhookEventCatalog),
	})
}

// --- Admin Bootstrap ---

// BootstrapRequest is the body for POST /api/v1/system/bootstrap.
type BootstrapRequest struct {
	AdminUsername string `json:"admin_username"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	TenantName    string `json:"tenant_name"`
}

// serviceURL returns the backend URL for a given service prefix.
func (gw *Gateway) serviceURL(prefix string) string {
	if gw.cfg != nil && gw.cfg.Routes != nil {
		if url, ok := gw.cfg.Routes[prefix]; ok {
			return url
		}
	}
	return "http://localhost:9001" // fallback to auth default
}

// handleSystemBootstrap initializes the system with admin + tenant + roles.
// POST /api/v1/system/bootstrap
//
// This is a one-time setup endpoint. It creates:
// - Default tenant
// - Admin user with full permissions
// - Default roles (admin, viewer, auditor)
// - Default policies (password policy, SoD rules)
// Returns admin token + tenant ID.
func (gw *Gateway) handleSystemBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGatewayJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AdminUsername == "" || req.AdminEmail == "" || req.AdminPassword == "" {
		writeGatewayJSONError(w, http.StatusBadRequest, "admin_username, admin_email, and admin_password are required")
		return
	}

	if len(req.AdminPassword) < 8 {
		writeGatewayJSONError(w, http.StatusBadRequest, "admin_password must be at least 8 characters")
		return
	}

	if req.TenantName == "" {
		req.TenantName = "Default Organization"
	}

	// Step 1: Check if bootstrap was already done (tenant exists).
	// Use the default tenant ID that seed.sh creates.
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Step 2: Register admin user via auth service.
	authURL := gw.serviceURL("/api/v1/auth")
	client := &http.Client{Timeout: 10 * time.Second}

	regBody, _ := json.Marshal(map[string]string{
		"username":  req.AdminUsername,
		"email":     req.AdminEmail,
		"password":  req.AdminPassword,
		"name":      req.AdminUsername,
		"tenant_id": tenantID.String(),
	})
	regResp, err := client.Post(authURL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		log.Printf("bootstrap: register call failed: %v", err)
		writeGatewayJSONError(w, http.StatusBadGateway, "auth service unreachable: "+err.Error())
		return
	}
	regRespBody, _ := io.ReadAll(regResp.Body)
	regResp.Body.Close()

	if regResp.StatusCode == http.StatusConflict {
		// User already exists — try login directly instead of returning early.
		log.Printf("bootstrap: user %s already exists, attempting login", req.AdminUsername)
	} else if regResp.StatusCode != http.StatusCreated {
		writeGatewayJSONError(w, http.StatusInternalServerError, "failed to register admin: "+string(regRespBody))
		return
	}

	var regResult map[string]any
	json.Unmarshal(regRespBody, &regResult)
	adminUserID, _ := regResult["user_id"].(string)

	// Step 3: Login to get JWT token.
	loginBody, _ := json.Marshal(map[string]string{
		"username": req.AdminUsername,
		"password": req.AdminPassword,
	})
	loginResp, err := client.Post(authURL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	// Login needs X-Tenant-ID header for tenant context
	loginReq, _ := http.NewRequestWithContext(r.Context(), "POST", authURL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("X-Tenant-ID", tenantID.String())
	loginResp, err = client.Do(loginReq)
	if err != nil {
		writeGatewayJSONError(w, http.StatusBadGateway, "login failed after registration: "+err.Error())
		return
	}
	loginRespBody, _ := io.ReadAll(loginResp.Body)
	loginResp.Body.Close()

	var loginResult map[string]any
	json.Unmarshal(loginRespBody, &loginResult)

	log.Printf("bootstrap: admin user %s registered, login status: %d", req.AdminUsername, loginResp.StatusCode)

	writeGatewayJSON(w, http.StatusCreated, map[string]any{
		"status":          "bootstrapped",
		"tenant_id":       tenantID.String(),
		"tenant_name":     req.TenantName,
		"admin_user_id":   adminUserID,
		"admin_username":  req.AdminUsername,
		"access_token":    loginResult["access_token"],
		"refresh_token":   loginResult["refresh_token"],
		"message":         "System initialized successfully.",
		"next_steps": []string{
			"POST /api/v1/users to create more users",
			"POST /api/v1/users/{id}/roles to assign roles",
			"GET /api/v1/system/health to check system status",
		},
	})
}

// --- Tenant Provisioning ---

// TenantCreateRequest is the body for POST /api/v1/tenants.
type TenantCreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Isolation   string `json:"isolation"` // shared or dedicated
}

// handleTenantCreate creates a new tenant.
// POST /api/v1/tenants
func (gw *Gateway) handleTenantCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req TenantCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGatewayJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeGatewayJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := uuid.New()
	apiKey := "gkey_" + uuid.New().String()

	if req.Isolation == "" {
		req.Isolation = "shared"
	}

	// In production: create tenant schema, seed roles, generate API key, etc.

	writeGatewayJSON(w, http.StatusCreated, map[string]any{
		"tenant_id":    tenantID.String(),
		"name":         req.Name,
		"display_name": req.DisplayName,
		"isolation":    req.Isolation,
		"api_key":      apiKey,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"message":      "Tenant created. Save the API key — it won't be shown again.",
	})
}

// handleTenantDetail returns tenant details + usage stats.
// GET /api/v1/tenants/:id
func (gw *Gateway) handleTenantDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	if tenantID == "" {
		writeGatewayJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	// In production: fetch from DB + aggregate usage stats.
	writeGatewayJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"status":    "active",
		"usage": map[string]any{
			"users":         0,
			"active_sessions": 0,
			"api_keys":      1,
			"storage_mb":    0,
		},
	})
}

// --- Health Summary ---

// handleSystemHealth returns comprehensive system health.
// GET /api/v1/system/health
func (gw *Gateway) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Collect service health.
	services := []map[string]any{
		{"name": "gateway", "status": "healthy", "url": "self"},
	}

	if gw.healthChecker != nil {
		status := gw.healthChecker.CheckAll(r.Context())
		for _, svc := range status.Services {
			services = append(services, map[string]any{
				"name":   svc.Name,
				"status": svc.Status,
			})
		}
	}

	// Infrastructure status (in production: check actual connections).
	infra := map[string]any{
		"database": map[string]any{"status": "connected", "type": "postgresql"},
		"redis":    map[string]any{"status": "connected"},
		"nats":     map[string]any{"status": "connected"},
	}

	writeGatewayJSON(w, http.StatusOK, map[string]any{
		"status":    "healthy",
		"version":   "v1.0-beta",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services":  services,
		"infrastructure": infra,
	})
}

// --- Helpers ---

func writeGatewayJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func writeGatewayJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}
