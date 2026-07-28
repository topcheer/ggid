package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// handleProviderStatus returns which auth providers are configured and available.
// This endpoint is public (no auth required) so login pages can determine
// which auth methods to show before user authentication.
//
// GET /api/v1/providers/status?tenant_id=<uuid>&client_id=<string>
//
// Response:
//   {"sms_otp": false, "email_otp": true, "passkey": true, "password": true}
func (h *Handler) handleProviderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var tenantID *uuid.UUID
	if tid := r.URL.Query().Get("tenant_id"); tid != "" {
		if parsed, err := uuid.Parse(tid); err == nil {
			tenantID = &parsed
		}
	}
	var clientID *string
	if cid := r.URL.Query().Get("client_id"); cid != "" {
		clientID = &cid
	}

	ctx := r.Context()
	pool := h.getPool()

	status := map[string]bool{
		"password":  true, // always available
		"passkey":   true, // passkey uses WebAuthn RP ID, always available if server has HTTPS
		"sms_otp":   false,
		"email_otp": false,
	}

	if pool != nil {
		// Check SMS provider in hierarchical config
		status["sms_otp"] = hierarchy.IsAvailable(ctx, pool, hierarchy.KeySMSProvider, tenantID, clientID)
		// If not in DB, check env vars
		if !status["sms_otp"] {
			status["sms_otp"] = os.Getenv("GGID_SMS_PROVIDER") != "" && os.Getenv("GGID_SMS_PROVIDER") != "log"
		}

		// Check Email provider in hierarchical config
		status["email_otp"] = hierarchy.IsAvailable(ctx, pool, hierarchy.KeyEmailProvider, tenantID, clientID)
		// If not in DB, check env vars
		if !status["email_otp"] {
			status["email_otp"] = os.Getenv("SMTP_HOST") != ""
		}
	} else {
		// No pool — fall back to env vars only
		status["sms_otp"] = os.Getenv("GGID_SMS_PROVIDER") != "" && os.Getenv("GGID_SMS_PROVIDER") != "log"
		status["email_otp"] = os.Getenv("SMTP_HOST") != ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getPool returns the pgxpool for DB queries, or nil if not configured.
func (h *Handler) getPool() *pgxpool.Pool {
	// Try to get pool from handler's pool field if it exists.
	// The auth service Handler struct has a pool field set at startup.
	if h.pool != nil {
		return h.pool
	}
	return nil
}
