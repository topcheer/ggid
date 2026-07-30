package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// POST /api/v1/oauth/device_authorize
func handleDeviceAuthorize(s *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := r.ParseForm(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid form data")
			return
		}
		clientID := r.FormValue("client_id")
		if clientID == "" {
			writeJSONError(w, http.StatusBadRequest, "client_id is required")
			return
		}
		scope := r.FormValue("scope")
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.FormValue("tenant_id")
		}
		// SECURITY (R22 P1): Fail-closed on missing/invalid tenant —
		// previously uuid.Nil was silently used, polluting JWT claims.
		tenantUUID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "valid tenant context required")
			return
		}

		var scopes []string
		if scope != "" {
			scopes = strings.Fields(scope)
		}

		entry, err := s.CreateDeviceAuthorization(&service.DeviceAuthorizationRequest{
			ClientID: clientID,
			TenantID: tenantUUID,
			Scope:    scopes,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		verificationURI := os.Getenv("CONSOLE_BASE_URL")
		if verificationURI == "" {
			verificationURI = "https://ggid-console.iot2.win"
		}
		verificationURI += "/device"
		writeJSON(w, http.StatusOK, map[string]any{
			"device_code":               entry.DeviceCode,
			"user_code":                 entry.UserCode,
			"verification_uri":          verificationURI,
			"verification_uri_complete": fmt.Sprintf("%s?user_code=%s", verificationURI, entry.UserCode),
			"expires_in":                900,
			"interval":                  5,
		})
	}
}

// POST /api/v1/oauth/device/verify
func handleDeviceVerify(s *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req struct {
			UserCode string `json:"user_code"`
			Action   string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			r.ParseForm()
			req.UserCode = r.FormValue("user_code")
			req.Action = r.FormValue("action")
		}
		if req.UserCode == "" {
			writeJSONError(w, http.StatusBadRequest, "user_code is required")
			return
		}
		if req.Action == "" {
			req.Action = "approve"
		}

		// SECURITY: user identity comes ONLY from the X-User-ID header set by
		// the gateway from the verified JWT — never from body/form fields.
		// Approving with uuid.Nil previously let anyone mint a token for the
		// all-zero subject in a tenant of their choosing (R-cron P1-2).
		userID, perr := uuid.Parse(r.Header.Get("X-User-ID"))
		if perr != nil || userID == uuid.Nil {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		var err error
		switch req.Action {
		case "approve":
			tenantID, tidErr := uuid.Parse(r.Header.Get("X-Tenant-ID"))
			if tidErr != nil {
				writeJSONError(w, http.StatusForbidden, "valid X-Tenant-ID header required")
				return
			}
			err = s.ApproveDeviceCode(req.UserCode, userID, tenantID)
		case "deny":
			err = s.DenyDeviceCode(req.UserCode)
		default:
			writeJSONError(w, http.StatusBadRequest, "action must be approve or deny")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": req.Action, "user_code": req.UserCode})
	}
}
