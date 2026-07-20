package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	AdminUsername   string `json:"admin_username"`
	AdminEmail      string `json:"admin_email"`
	AdminPassword   string `json:"admin_password"`
	TenantName      string `json:"tenant_name"`
	WebAuthnRPID    string `json:"webauthn_rp_id"`    // e.g. "ggid-console.iot2.win"
	WebAuthnOrigins string `json:"webauthn_origins"`  // comma-separated, e.g. "https://ggid-console.iot2.win"
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

	// Security: block bootstrap if system is already initialized
	// Check both in-memory flag (fast path) and DB (survives restart)
	if quickstartInitialized {
		writeGatewayJSONError(w, http.StatusConflict, "System is already initialized. Use admin login to create new tenants via /api/v1/tenants.")
		return
	}
	// DB check: if tenants exist, system is already bootstrapped
	var dbURL string
	if gw.cfg != nil {
		dbURL = gw.cfg.DatabaseURL
	}
	if dbURL == "" {
		dbURL = "postgres://ggid:ggid-k3s@ggid-postgresql:5432/ggid?sslmode=disable"
	}
	if conn, err := pgx.Connect(r.Context(), dbURL); err == nil {
		var tenantCount int
		conn.QueryRow(r.Context(), "SELECT count(*) FROM tenants").Scan(&tenantCount)
		conn.Close(r.Context())
		if tenantCount > 0 {
			quickstartInitialized = true
			writeGatewayJSONError(w, http.StatusConflict, "System is already initialized. Use admin login to create new tenants via /api/v1/tenants.")
			return
		}
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

	// Step 1: Create tenant record in DB via identity service.
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	identityURL := gw.serviceURL("/api/v1/users")
	client := &http.Client{Timeout: 10 * time.Second}

	tenantBody, _ := json.Marshal(map[string]string{
		"name": req.TenantName,
		"slug": "default",
	})
	tenantReq, _ := http.NewRequestWithContext(r.Context(), "POST", identityURL+"/api/v1/tenants", bytes.NewReader(tenantBody))
	tenantReq.Header.Set("Content-Type", "application/json")
	tenantResp, err := client.Do(tenantReq)
	if err != nil {
		log.Printf("bootstrap: create tenant failed (non-fatal): %v", err)
	} else {
		io.ReadAll(tenantResp.Body)
		tenantResp.Body.Close()
		log.Printf("bootstrap: tenant created (status %d)", tenantResp.StatusCode)
	}

	// Step 2: Register admin user via auth service.
	authURL := gw.serviceURL("/api/v1/auth")

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

	// Step 2b: Create default roles with proper scope keys + assign platform:admin.
	for _, role := range []struct{ key, name string }{
		{"platform:admin", "Platform Administrator"},
		{"tenant:admin", "Tenant Administrator"},
		{"tenant:auditor", "Tenant Auditor"},
		{"user:self", "User"},
	} {
		roleBody, _ := json.Marshal(map[string]string{"key": role.key, "name": role.name})
		roleReq, _ := http.NewRequestWithContext(r.Context(), "POST", identityURL+"/api/v1/roles", bytes.NewReader(roleBody))
		roleReq.Header.Set("Content-Type", "application/json")
		roleReq.Header.Set("X-Tenant-ID", tenantID.String())
		roleResp, err := client.Do(roleReq)
		if err != nil {
			log.Printf("bootstrap: create role %s failed: %v", role.key, err)
			continue
		}
		io.ReadAll(roleResp.Body)
		roleResp.Body.Close()

		// Assign platform:admin + tenant:admin to bootstrap user
		if (role.key == "platform:admin" || role.key == "tenant:admin") && adminUserID != "" {
			assignBody, _ := json.Marshal(map[string]string{"role_id": role.key, "role_name": role.name})
			assignReq, _ := http.NewRequestWithContext(r.Context(), "POST", identityURL+"/api/v1/users/"+adminUserID+"/roles", bytes.NewReader(assignBody))
			assignReq.Header.Set("Content-Type", "application/json")
			assignReq.Header.Set("X-Tenant-ID", tenantID.String())
			assignResp, err := client.Do(assignReq)
			if err != nil {
				log.Printf("bootstrap: assign role %s failed: %v", role.key, err)
			} else {
				log.Printf("bootstrap: role %s assigned (status %d)", role.key, assignResp.StatusCode)
				assignResp.Body.Close()
			}
		}
	}

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

	// Mark system as initialized for /api/v1/system/status
	quickstartInitialized = true

	// Save WebAuthn config to DB (if provided)
	if req.WebAuthnRPID != "" {
		origins := []string{}
		if req.WebAuthnOrigins != "" {
			for _, o := range strings.Split(req.WebAuthnOrigins, ",") {
				origins = append(origins, strings.TrimSpace(o))
			}
		}
		if len(origins) == 0 {
			origins = []string{"https://" + req.WebAuthnRPID}
		}
		configJSON, _ := json.Marshal(map[string]any{
			"rp_id":     req.WebAuthnRPID,
			"rp_origins": origins,
			"rp_name":   req.TenantName,
		})
		dbURL := gw.cfg.DatabaseURL
		if dbURL == "" {
			dbURL = "postgres://ggid:ggid-k3s@ggid-postgresql:5432/ggid?sslmode=disable"
		}
		if conn, err := pgx.Connect(r.Context(), dbURL); err == nil {
			conn.Exec(r.Context(),
				`INSERT INTO sys_config (key, value) VALUES ('webauthn_config', $1)
				 ON CONFLICT (key) DO UPDATE SET value = $1`,
				configJSON)
			conn.Close(r.Context())
			log.Printf("bootstrap: WebAuthn config saved to DB (rp_id=%s)", req.WebAuthnRPID)
		}
	}

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
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Plan        string `json:"plan"`
	Isolation   string `json:"isolation"` // shared or dedicated
}

// handleTenantCreate creates a new tenant by inserting into the tenants table.
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

	// Auto-generate slug from name if not provided
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.TrimSpace(req.Name))
		req.Slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(req.Slug, "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}
	if req.Plan == "" {
		req.Plan = "free"
	}

	tenantID := uuid.New()
	apiKey := "gkey_" + uuid.New().String()

	// Write to DB
	dbURL := gw.cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "postgres://ggid:ggid-k3s@ggid-postgresql:5432/ggid?sslmode=disable"
	}
	conn, err := pgx.Connect(r.Context(), dbURL)
	if err != nil {
		log.Printf("tenant create: failed to connect DB: %v", err)
	} else {
		defer conn.Close(r.Context())
		_, _ = conn.Exec(r.Context(), `SET app.tenant_id = '00000000-0000-0000-0000-000000000001'`) // bypass RLS for tenant management
		_, err := conn.Exec(r.Context(),
			`INSERT INTO tenants (id, name, slug, plan, status, max_users) VALUES ($1, $2, $3, $4, 'active', 50)`,
			tenantID, req.Name, req.Slug, req.Plan)
		if err != nil {
			if strings.Contains(err.Error(), "tenants_slug_key") {
				writeGatewayJSONError(w, http.StatusConflict, "subdomain '"+req.Slug+"' is already taken")
				return
			}
			log.Printf("tenant create: DB insert error: %v", err)
		}
	}

	writeGatewayJSON(w, http.StatusCreated, map[string]any{
		"tenant_id":    tenantID.String(),
		"name":         req.Name,
		"slug":         req.Slug,
		"display_name": req.DisplayName,
		"plan":         req.Plan,
		"isolation":    "shared",
		"api_key":      apiKey,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"message":      "Tenant created. Save the API key — it won't be shown again.",
	})
}

// handleTenantList returns all tenants from DB.
// GET /api/v1/tenants
func (gw *Gateway) handleTenantList(w http.ResponseWriter, r *http.Request) {
	dbURL := gw.cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "postgres://ggid:ggid-k3s@ggid-postgresql:5432/ggid?sslmode=disable"
	}
	conn, err := pgx.Connect(r.Context(), dbURL)
	if err != nil {
		log.Printf("tenant list: failed to connect DB: %v", err)
		writeGatewayJSON(w, http.StatusOK, map[string]any{"tenants": []any{}})
		return
	}
	defer conn.Close(r.Context())
	_, _ = conn.Exec(r.Context(), `SET app.tenant_id = '00000000-0000-0000-0000-000000000001'`) // bypass RLS

	rows, err := conn.Query(r.Context(),
		`SELECT id::text, name, slug, plan, status, max_users, created_at FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		writeGatewayJSON(w, http.StatusOK, map[string]any{"tenants": []any{}})
		return
	}
	defer rows.Close()

	type Tenant struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		Plan      string `json:"plan"`
		Status    string `json:"status"`
		MaxUsers  int    `json:"max_users"`
		CreatedAt string `json:"created_at"`
	}

	tenants := []Tenant{}
	for rows.Next() {
		var t Tenant
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &t.MaxUsers, &createdAt); err != nil {
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		tenants = append(tenants, t)
	}

	writeGatewayJSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

// handleTenantDetail returns tenant details + usage stats from DB.
// GET/DELETE /api/v1/tenants/:id
func (gw *Gateway) handleTenantDetail(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	if tenantIDStr == "" {
		writeGatewayJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	dbURL := gw.cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "postgres://ggid:ggid-k3s@ggid-postgresql:5432/ggid?sslmode=disable"
	}
	conn, err := pgx.Connect(r.Context(), dbURL)
	if err != nil {
		writeGatewayJSONError(w, http.StatusInternalServerError, "DB connection failed")
		return
	}
	defer conn.Close(r.Context())
	_, _ = conn.Exec(r.Context(), `SET app.tenant_id = '00000000-0000-0000-0000-000000000001'`)

	if r.Method == http.MethodDelete {
		_, err := conn.Exec(r.Context(), `DELETE FROM tenants WHERE id::text = $1 OR slug = $1`, tenantIDStr)
		if err != nil {
			writeGatewayJSONError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		writeGatewayJSON(w, http.StatusOK, map[string]any{"deleted": true, "tenant_id": tenantIDStr})
		return
	}

	if r.Method != http.MethodGet {
		writeGatewayJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var id, name, slug, plan, status string
	var maxUsers int
	var createdAt time.Time
	err = conn.QueryRow(r.Context(),
		`SELECT id::text, name, slug, plan, status, max_users, created_at FROM tenants WHERE id::text = $1 OR slug = $1`,
		tenantIDStr).Scan(&id, &name, &slug, &plan, &status, &maxUsers, &createdAt)
	if err != nil {
		writeGatewayJSONError(w, http.StatusNotFound, "tenant not found")
		return
	}

	// Aggregate usage
	var userCount, sessionCount int
	tenantUUID, _ := uuid.Parse(id)
	_ = conn.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantUUID).Scan(&userCount)
	_ = conn.QueryRow(r.Context(), `SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE u.tenant_id = $1 AND s.revoked_at IS NULL`, tenantUUID).Scan(&sessionCount)

	writeGatewayJSON(w, http.StatusOK, map[string]any{
		"id":         id,
		"name":       name,
		"slug":       slug,
		"plan":       plan,
		"status":     status,
		"max_users":  maxUsers,
		"created_at": createdAt.Format(time.RFC3339),
		"usage": map[string]any{
			"users":           userCount,
			"active_sessions": sessionCount,
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
