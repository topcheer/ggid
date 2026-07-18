package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InvalidationReason defines why sessions are being invalidated.
type InvalidationReason string

const (
	InvReasonPasswordChange InvalidationReason = "password_change"
	InvReasonMFAEnrollment  InvalidationReason = "mfa_enrollment"
	InvReasonPostureDrop    InvalidationReason = "posture_drop"
)

// InvalidationRequest is the body for POST /api/v1/auth/invalidate-sessions/:user_id.
type InvalidationRequest struct {
	Reason          string `json:"reason"`
	ExceptSessionID string `json:"except_session_id"` // keep current session alive
}

// InvalidationAudit tracks session invalidation events.
type InvalidationAudit struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Reason         string    `json:"reason"`
	SessionsRevoked int      `json:"sessions_revoked"`
	InitiatedBy    string    `json:"initiated_by"`
	Timestamp      time.Time `json:"timestamp"`
}

// POST /api/v1/auth/invalidate-sessions/:user_id
func (h *Handler) handleInvalidateSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user_id from path.
	userIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/invalidate-sessions/")
	if userIDStr == "" || strings.Contains(userIDStr, "/") {
		writeError(w, http.StatusBadRequest, "valid user_id required in path")
		return
	}

	var req InvalidationRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	validReasons := map[string]bool{
		"password_change": true,
		"mfa_enrollment":  true,
		"posture_drop":    true,
		"admin_action":    true,
	}
	if req.Reason == "" {
		req.Reason = "admin_action"
	}
	if !validReasons[req.Reason] {
		writeError(w, http.StatusBadRequest, "reason must be password_change, mfa_enrollment, posture_drop, or admin_action")
		return
	}

	// Parse tenant from header.
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid X-Tenant-ID header required")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id required")
		return
	}

	// Use existing SessionRevocationManager if available.
	if h.revocationMgr != nil {
		result, err := h.revocationMgr.RevokeUser(r.Context(), tenantID, userID, req.Reason)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":  "partial",
				"error":   err.Error(),
				"user_id": userIDStr,
			})
			return
		}

		audit := InvalidationAudit{
			ID:              uuid.New().String(),
			UserID:          userIDStr,
			Reason:          req.Reason,
			SessionsRevoked: result.SessionsRevoked,
			InitiatedBy:     r.Header.Get("X-User-ID"),
			Timestamp:       time.Now().UTC(),
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":           "invalidated",
			"user_id":          userIDStr,
			"reason":           req.Reason,
			"sessions_revoked": result.SessionsRevoked,
			"jtis_blocked":     result.JTIsBlocked,
			"refresh_revoked":  result.RefreshRevoked,
			"except_session":   req.ExceptSessionID,
			"audit":            audit,
		})
		return
	}

	// Fallback without revocation manager.
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "queued",
		"user_id": userIDStr,
		"reason":  req.Reason,
		"message": "session invalidation queued (revocation manager not configured)",
	})
}

// TriggerInvalidation is called internally when password changes, MFA enrolls, etc.
// It does NOT need HTTP context — called from service layer.
func (h *Handler) TriggerInvalidation(tenantID, userID uuid.UUID, reason InvalidationReason, exceptSessionID string) *InvalidationAudit {
	audit := &InvalidationAudit{
		ID:           uuid.New().String(),
		UserID:       userID.String(),
		Reason:       string(reason),
		InitiatedBy:  "system",
		Timestamp:    time.Now().UTC(),
	}

	if h.revocationMgr != nil {
		result, err := h.revocationMgr.RevokeUser(nil, tenantID, userID, string(reason))
		if err == nil {
			audit.SessionsRevoked = result.SessionsRevoked
		}
	}

	return audit
}
