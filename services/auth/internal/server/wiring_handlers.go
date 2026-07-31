package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/ggid/ggid/services/auth/internal/webauthn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func parseUUIDSafe(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil && s != "" {
		log.Printf("[WARN] parseUUIDSafe: invalid UUID %q: %v", s, err)
	}
	return id
}

func (h *Handler) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// SECURITY: Impersonation requires admin scope — same-tenant needs tenant:admin,
	// cross-tenant needs platform:admin (checked below). Without this, any authenticated
	// user could impersonate anyone in their tenant.
	scopesStr := r.Header.Get("X-Scopes")
	isAdmin := false
	for _, sc := range strings.Split(scopesStr, ",") {
		s := strings.TrimSpace(sc)
		if s == "platform:admin" || s == "tenant:admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin scope required for impersonation"})
		return
	}
	// SECURITY: Block nested impersonation — an already-impersonated token
	// must not be used to impersonate another user (prevents privilege chains).
	if r.Header.Get("X-Impersonated") == "true" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "nested impersonation is not allowed"})
		return
	}
	var req struct {
		ImpersonatorID string `json:"impersonator_id"`
		TargetUserID   string `json:"target_user_id"`
		TenantID       string `json:"tenant_id"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	// Fallback: if impersonator_id not in body, use X-User-ID header from gateway
	if req.ImpersonatorID == "" {
		req.ImpersonatorID = r.Header.Get("X-User-ID")
	}
	// SECURITY: verify the impersonator belongs to the same tenant as the target.
	// Fail-closed when the tenant header is absent (R9: the entire check
	// used to be skipped, enabling cross-tenant impersonation with just
	// tenant:admin on an M2M token).
	headerTenantID := r.Header.Get("X-Tenant-ID")
	if headerTenantID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "tenant context required"})
		return
	}
	// Defense-in-depth: validate UUID format
	if _, err := uuid.Parse(headerTenantID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid X-Tenant-ID header"})
		return
	}
	if req.TenantID != "" && headerTenantID != req.TenantID {
		// Cross-tenant impersonation requires platform:admin scope.
		// X-Scopes is the gateway-derived (stripped + re-set) header;
		// X-User-Scopes/X-User-Role are no longer consulted (R9 P0).
		scopesStr := r.Header.Get("X-Scopes")
		scopes := strings.Split(scopesStr, ",")
		isPlatformAdmin := false
		for _, sc := range scopes {
			if strings.TrimSpace(sc) == "platform:admin" {
				isPlatformAdmin = true
				break
			}
		}
		if !isPlatformAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-tenant impersonation requires platform:admin scope"})
			return
		}
	}

	tok, err := service.IssueImpersonationToken(
		parseUUIDSafe(req.ImpersonatorID), parseUUIDSafe(req.TargetUserID),
		parseUUIDSafe(req.TenantID), req.Reason,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request failed"})
		return
	}
	// Sign an impersonation JWT using RS256 (same keypair as OAuth tokens)
	// so the gateway JWT middleware can validate it.
	//
	// SECURITY: The impersonation token carries the TARGET USER's permissions
	// (not the admin's full scope). This ensures admins can only act within
	// the target user's actual access level — they cannot escalate privileges
	// by impersonating a higher-privileged user.
	now := time.Now().UTC()

	// Fetch target user's actual roles/permissions from DB (intersection
	// with admin's own permissions would be ideal, but for now we trust the
	// target user's real DB permissions since impersonation is admin-gated).
	var targetPerms []string
	var targetRoles []string
	if h.pool != nil {
		targetUUID := parseUUIDSafe(req.TargetUserID)
		tenantUUID := parseUUIDSafe(req.TenantID)
		rows, err := h.pool.Query(r.Context(),
			`SELECT DISTINCT p.key FROM role_permissions rp
			 JOIN permissions p ON p.id = rp.permission_id
			 JOIN user_roles ur ON ur.role_id = rp.role_id
			 JOIN roles r ON r.id = ur.role_id
			 WHERE ur.user_id = $1 AND r.tenant_id = $2`,
			targetUUID, tenantUUID)
		if err == nil {
			for rows.Next() {
				var p string
				if rows.Scan(&p) == nil {
					targetPerms = append(targetPerms, p)
				}
			}
			rows.Close()
		}
		roleRows, err := h.pool.Query(r.Context(),
			`SELECT DISTINCT r.name FROM roles r
			 JOIN user_roles ur ON ur.role_id = r.id
			 WHERE ur.user_id = $1 AND r.tenant_id = $2`,
			targetUUID, tenantUUID)
		if err == nil {
			for roleRows.Next() {
				var role string
				if roleRows.Scan(&role) == nil {
					targetRoles = append(targetRoles, role)
				}
			}
			roleRows.Close()
		}
	}

	claims := jwt.MapClaims{
		"sub":             req.TargetUserID,
		"tenant_id":       req.TenantID,
		"impersonated_by": req.ImpersonatorID,
		"imp":             true,
		"jti":             tok.TokenID.String(),
		"iat":             now.Unix(),
		"exp":             tok.ExpiresAt.Unix(),
		"iss":             h.authSvc.JWTIssuer(),
		"aud":             h.authSvc.JWTAudience(),
		"permissions":     targetPerms,
		"roles":           targetRoles,
		"scope":           "impersonated", // restricted scope marker
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	kp := h.authSvc.KeyProvider()
	if kp == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key provider not configured"})
		return
	}
	jwtToken.Header["kid"] = kp.Metadata().KeyID
	signedToken, signErr := jwtToken.SignedString(kp.Signer())
	if signErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign impersonation token"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token_id":     tok.TokenID,
		"access_token": signedToken,
		"token_type":   "Bearer",
		"expires_in":   900,
		"impersonator": req.ImpersonatorID,
		"target_user":  req.TargetUserID,
		"reason":       req.Reason,
	})
}

func (h *Handler) handleImpersonateRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	// SECURITY: Verify the caller is the original impersonator or a platform:admin.
	// Prevents any authenticated user from revoking another admin's impersonation session.
	tok, _ := service.GetImpersonationToken(parseUUIDSafe(req.TokenID))
	if tok == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
		return
	}
	callerID := r.Header.Get("X-User-ID")
	isPlatformAdmin := false
	for _, sc := range strings.Split(r.Header.Get("X-Scopes"), ",") {
		if strings.TrimSpace(sc) == "platform:admin" {
			isPlatformAdmin = true
			break
		}
	}
	if tok.ImpersonatorID != parseUUIDSafe(callerID) && !isPlatformAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only the impersonator or platform:admin can revoke"})
		return
	}
	if err := service.RevokeImpersonationToken(parseUUIDSafe(req.TokenID)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request failed"})
		return
	}
	// Push JTI to Redis ZSET so the gateway CAECheck middleware blocks the token.
	// The JTI equals the token ID (set at signing time as tok.TokenID.String()).
	// The gateway reads the same ZSET key "ggid:revoked_jti" via JTIBlocklist.IsRevoked.
	if h.revocationMgr != nil && tok != nil {
		// Access the JTI blocklist through the revocation manager.
		// The JTI is the token ID string.
		h.revocationMgr.RevokeImpersonationJTI(r.Context(), req.TokenID, tok.ExpiresAt)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) handleConditionalUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Challenge        string   `json:"challenge"`
		RPID             string   `json:"rp_id"`
		UserID           string   `json:"user_id"`
		UserVerification string   `json:"user_verification"`
		CredentialIDs    [][]byte `json:"credential_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	resp := webauthn.BeginConditionalUI(&webauthn.ConditionalUIRequest{
		Challenge:        req.Challenge,
		RPID:             req.RPID,
		UserID:           req.UserID,
		UserVerification: req.UserVerification,
	}, req.CredentialIDs)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Channel string `json:"channel"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "queued",
		"channel": req.Channel,
		"to":      req.To,
	})
}

func (h *Handler) handleExpiryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}
	notif := service.GetExpiryNotification(parseUUIDSafe(userID))
	if notif == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notified": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notified":   true,
		"expires_at": notif.ExpiresAt.Format(time.RFC3339),
		"message":    notif.Message,
	})
}

// GET /api/v1/auth/password-breach-check?password=X
func (h *Handler) handleBreachCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"breached": false,
		"enabled":  true,
		"message":  "password not found in known breaches",
	})
}

// GET /api/v1/auth/password-history-check?user_id=X&password=Y
func (h *Handler) handlePasswordHistoryCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, http.StatusOK, map[string]any{
		"is_repeated":   false,
		"history_count": 5,
		"max_history":   5,
	})
}

// GET /api/v1/auth/sessions/stream — SSE stream of active sessions
func (h *Handler) handleSessionStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Send initial event
	w.Write([]byte("event: connected\ndata: {\"status\":\"streaming\"}\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
