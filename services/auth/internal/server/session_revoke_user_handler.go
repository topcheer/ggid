package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// SessionRevokeUserRequest is the body for POST /api/v1/auth/sessions/revoke-user.
type SessionRevokeUserRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Reason   string `json:"reason"`
}

// handleRevokeUser revokes all sessions for a user (admin endpoint, JWT auth required).
// POST /api/v1/auth/sessions/revoke-user
//
// Performs multi-layer revocation:
//   - DB sessions marked revoked
//   - Redis JTI blocklist updated (gateway CAECheck will 401 revoked tokens)
//   - Refresh tokens revoked
//   - Audit event published
func (h *Handler) handleRevokeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SessionRevokeUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
			tenantID, err = uuid.Parse(tidStr)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "valid tenant_id is required")
			return
		}
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id is required")
		return
	}

	if req.Reason == "" {
		req.Reason = "admin_revocation"
	}

	if h.revocationMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "no_active_sessions", "revoked": 0})
		return
	}

	result, err := h.revocationMgr.RevokeUser(r.Context(), tenantID, userID, req.Reason)
	if err != nil {
		slog.Error("revoke-user error", "error", err)
		writeError(w, http.StatusInternalServerError, "revocation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "revoked",
		"sessions_revoked": result.SessionsRevoked,
		"jtis_blocked":     result.JTIsBlocked,
		"refresh_revoked":  result.RefreshRevoked,
	})
}

// handleInternalRevokeUser is the InternalAuth-protected endpoint for cross-service
// session revocation (e.g. audit/ITDR service triggering revocation via NATS).
// POST /api/v1/auth/internal/revoke-user
func (h *Handler) handleInternalRevokeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Verify internal auth HMAC if configured.
	if h.internalSecret != "" {
		if !h.verifyInternalSignature(r) {
			writeError(w, http.StatusUnauthorized, "invalid internal auth signature")
			return
		}
	}

	var req SessionRevokeUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid tenant_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id is required")
		return
	}

	if req.Reason == "" {
		req.Reason = "internal_revocation"
	}

	if h.revocationMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "no_active_sessions", "revoked": 0})
		return
	}

	result, err := h.revocationMgr.RevokeUser(r.Context(), tenantID, userID, req.Reason)
	if err != nil {
		slog.Error("internal revoke-user error", "error", err)
		writeError(w, http.StatusInternalServerError, "revocation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "revoked",
		"sessions_revoked": result.SessionsRevoked,
		"jtis_blocked":     result.JTIsBlocked,
		"refresh_revoked":  result.RefreshRevoked,
	})
}

// verifyInternalSignature checks the HMAC-SHA256 signature on an internal request.
// Uses current secret, with fallback to previous secret for key rotation.
// Replay protection: rejects timestamps older than 120 seconds.
func (h *Handler) verifyInternalSignature(r *http.Request) bool {
	sig := r.Header.Get("X-Internal-Signature")
	ts := r.Header.Get("X-Internal-Timestamp")
	bodyHash := r.Header.Get("X-Internal-Body-Hash")
	if sig == "" || ts == "" {
		return false
	}

	// Replay protection: reject timestamps > 120s old.
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(tsInt, 0)) > 120*time.Second {
		return false
	}

	// Verify against current secret.
	if matchHMAC(r.URL.Path, ts, bodyHash, h.internalSecret, sig) {
		return true
	}

	// Fallback to previous secret during rotation.
	if h.internalPrevSecret != "" {
		return matchHMAC(r.URL.Path, ts, bodyHash, h.internalPrevSecret, sig)
	}

	return false
}

// matchHMAC computes HMAC-SHA256(method + path + timestamp + bodyHash, secret) and compares.
func matchHMAC(path, ts, bodyHash, secret, expected string) bool {
	message := "POST:" + path + ":" + ts + ":" + bodyHash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	computed := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(expected))
}
