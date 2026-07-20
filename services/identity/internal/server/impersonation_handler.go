package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- Tenant Access Consent ---

// handleConsentCRUD handles:
//   POST   /api/v1/tenants/{id}/access/grant   — grant consent
//   DELETE /api/v1/tenants/{id}/access/{cid}   — revoke consent
//   GET    /api/v1/tenants/{id}/access         — list consents
func (h *HTTPHandler) handleConsentCRUD(w http.ResponseWriter, r *http.Request) {
	pool := h.svc.Pool()
	if pool == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database not available"})
		return
	}

	// Parse path: /api/v1/tenants/{id}/access[/{consent_id}]
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// parts: [api v1 tenants {id} access ...]

	if len(parts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}

	tenantID := parts[3]
	action := parts[4] // "access"

	if action != "access" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected /access path"})
		return
	}

	// Sub-path: /access/grant, /access/{consent_id}, or just /access
	subPath := ""
	if len(parts) >= 6 {
		subPath = parts[5]
	}

	switch {
	case r.Method == http.MethodPost && subPath == "grant":
		h.consentGrant(w, r, tenantID)
	case r.Method == http.MethodDelete && subPath != "":
		h.consentRevoke(w, r, tenantID, subPath)
	case r.Method == http.MethodGet:
		h.consentList(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandler) consentGrant(w http.ResponseWriter, r *http.Request, tenantID string) {
	// Parse tenantID to UUID
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
		return
	}

	var req struct {
		GrantedTo string `json:"granted_to"`
		Scope     string `json:"scope"`
		ExpiresAt string `json:"expires_at"` // ISO 8601, optional
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	if req.GrantedTo == "" {
		req.GrantedTo = "platform_admin"
	}
	if req.Scope == "" {
		req.Scope = "support"
	}
	if req.Scope != "support" && req.Scope != "audit" && req.Scope != "full" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scope must be support, audit, or full"})
		return
	}

	// Parse optional expiry
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid expires_at format"})
			return
		}
		expiresAt = &t
	}

	// Get granted_by from X-User-ID header (set by gateway)
	grantedByStr := r.Header.Get("X-User-ID")
	grantedBy, err := uuid.Parse(grantedByStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var consentID string
	if expiresAt != nil {
		err = pool_QueryRow(h, r, `
			INSERT INTO tenant_access_consents (tenant_id, granted_to, granted_by, scope, expires_at, reason)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id::text`, tid, req.GrantedTo, grantedBy, req.Scope, *expiresAt, req.Reason).Scan(&consentID)
	} else {
		err = pool_QueryRow(h, r, `
			INSERT INTO tenant_access_consents (tenant_id, granted_to, granted_by, scope, reason)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id::text`, tid, req.GrantedTo, grantedBy, req.Scope, req.Reason).Scan(&consentID)
	}
	if err != nil {
		slog.Error("consent grant: DB error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create consent"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"consent_id": consentID,
		"id":         consentID,
		"tenant_id":  tenantID,
		"granted_to": req.GrantedTo,
		"scope":      req.Scope,
		"status":     "active",
	})
}

func (h *HTTPHandler) consentRevoke(w http.ResponseWriter, r *http.Request, tenantID, consentID string) {
	tag, err := h.svc.Pool().Exec(r.Context(), `
		UPDATE tenant_access_consents SET revoked_at = NOW()
		WHERE id::text = $1 AND tenant_id::text = $2 AND revoked_at IS NULL`,
		consentID, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "consent not found or already revoked"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "consent_id": consentID})
}

func (h *HTTPHandler) consentList(w http.ResponseWriter, r *http.Request, tenantID string) {
	rows, err := h.svc.Pool().Query(r.Context(), `
		SELECT id::text, tenant_id::text, granted_to, granted_by::text,
		       scope, COALESCE(expires_at::text, ''), COALESCE(revoked_at::text, ''),
		       COALESCE(reason, ''), created_at
		FROM tenant_access_consents
		WHERE tenant_id::text = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query consents"})
		return
	}
	defer rows.Close()

	consents := []map[string]any{}
	for rows.Next() {
		var id, tenantIDStr, grantedTo, grantedBy, scope, expiresAt, revokedAt, reason string
		var createdAt time.Time
		if err := rows.Scan(&id, &tenantIDStr, &grantedTo, &grantedBy, &scope, &expiresAt, &revokedAt, &reason, &createdAt); err != nil {
			continue
		}
		status := "active"
		if revokedAt != "" {
			status = "revoked"
		} else if expiresAt != "" {
			exp, _ := time.Parse(time.RFC3339, expiresAt)
			if time.Now().After(exp) {
				status = "expired"
			}
		}
		consents = append(consents, map[string]any{
			"id":         id,
			"consent_id": id,
			"tenant_id":  tenantIDStr,
			"granted_to": grantedTo,
			"granted_by": grantedBy,
			"scope":      scope,
			"expires_at": expiresAt,
			"revoked_at": revokedAt,
			"reason":     reason,
			"status":     status,
			"created_at": createdAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"consents": consents,
		"total":    len(consents),
	})
}

// --- Impersonation ---

// handleImpersonation handles:
//   POST /api/v1/impersonate/start  — start impersonation
//   POST /api/v1/impersonate/end    — end impersonation
//   GET  /api/v1/impersonate/active — list active sessions
func (h *HTTPHandler) handleImpersonation(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	case path == "/api/v1/impersonate/start" && r.Method == http.MethodPost:
		h.impersonateStart(w, r)
	case path == "/api/v1/impersonate/end" && r.Method == http.MethodPost:
		h.impersonateEnd(w, r)
	case path == "/api/v1/impersonate/active" && r.Method == http.MethodGet:
		h.impersonateActive(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandler) impersonateStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		TargetUserID string `json:"target_user_id"` // optional
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	if req.TenantID == "" || req.Reason == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and reason are required"})
		return
	}

	impersonatorStr := r.Header.Get("X-User-ID")
	impersonatorID, err := uuid.Parse(impersonatorStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
		return
	}

	// Check for active consent
	var consentID string
	var scope string
	err = h.svc.Pool().QueryRow(r.Context(), `
		SELECT id::text, scope FROM tenant_access_consents
		WHERE tenant_id = $1 AND granted_to IN ('platform_admin', $2)
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC LIMIT 1`, tenantUUID, impersonatorStr).Scan(&consentID, &scope)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":  "no active consent for this tenant",
			"action": "request_consent",
			"message": "Tenant admin must grant access first",
		})
		return
	}

	// End any existing active impersonation for this admin+tenant
	_, _ = h.svc.Pool().Exec(r.Context(), `
		UPDATE impersonation_sessions SET ended_at = NOW()
		WHERE impersonator_id = $1 AND tenant_id = $2 AND ended_at IS NULL`,
		impersonatorID, tenantUUID)

	// Create new impersonation session
	var sessionID string
	err = h.svc.Pool().QueryRow(r.Context(), `
		INSERT INTO impersonation_sessions (tenant_id, impersonator_id, consent_id, reason, scope, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text`,
		tenantUUID, impersonatorID, consentID, req.Reason, scope,
		r.RemoteAddr, r.UserAgent()).Scan(&sessionID)
	if err != nil {
		slog.Error("impersonate start: DB error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to start impersonation"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":      sessionID,
		"tenant_id":       req.TenantID,
		"scope":           scope,
		"consent_id":      consentID,
		"status":          "active",
		"message":         "Impersonation started. All actions are audited.",
	})
}

func (h *HTTPHandler) impersonateEnd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id required"})
		return
	}

	tag, err := h.svc.Pool().Exec(r.Context(), `
		UPDATE impersonation_sessions SET ended_at = NOW()
		WHERE id::text = $1 AND ended_at IS NULL`, req.SessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to end session"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found or already ended"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ended": true, "session_id": req.SessionID})
}

func (h *HTTPHandler) impersonateActive(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.Pool().Query(r.Context(), `
		SELECT s.id::text, s.tenant_id::text, s.impersonator_id::text,
		       COALESCE(s.target_user_id::text, ''), s.scope, s.reason,
		       s.started_at, COALESCE(s.ip_address::text, ''), COALESCE(s.user_agent, ''),
		       t.name as tenant_name
		FROM impersonation_sessions s
		LEFT JOIN tenants t ON t.id = s.tenant_id
		WHERE s.ended_at IS NULL
		ORDER BY s.started_at DESC`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query sessions"})
		return
	}
	defer rows.Close()

	sessions := []map[string]any{}
	for rows.Next() {
		var id, tenantID, impersonatorID, targetUserID, scope, reason string
		var startedAt time.Time
		var ipAddress, userAgent, tenantName string
		if err := rows.Scan(&id, &tenantID, &impersonatorID, &targetUserID, &scope, &reason, &startedAt, &ipAddress, &userAgent, &tenantName); err != nil {
			continue
		}
		sessions = append(sessions, map[string]any{
			"session_id":      id,
			"tenant_id":       tenantID,
			"tenant_name":     tenantName,
			"impersonator_id": impersonatorID,
			"target_user_id":  targetUserID,
			"scope":           scope,
			"reason":          reason,
			"started_at":      startedAt,
			"ip_address":      ipAddress,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// pool_QueryRow is a helper to access the identity service pool.
// Used by consentGrant to avoid duplicating the nil check logic.
type poolHelper struct{ h *HTTPHandler }

func pool_QueryRow(h *HTTPHandler, r *http.Request, query string, args ...any) rowScanner {
	return h.svc.Pool().QueryRow(r.Context(), query, args...)
}

type rowScanner interface {
	Scan(dest ...any) error
}
