package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/ggid/ggid/services/oauth/internal/service"
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
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = r.FormValue("tenant_id")
		}

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.FormValue("tenant_id")
		}
		tenantUUID, _ := uuid.Parse(tenantIDStr)

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
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		verificationURI := "https://ggid-console.iot2.win/device"
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
			UserID   string `json:"user_id"`
			TenantID string `json:"tenant_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			r.ParseForm()
			req.UserCode = r.FormValue("user_code")
			req.Action = r.FormValue("action")
			req.UserID = r.FormValue("user_id")
			req.TenantID = r.FormValue("tenant_id")
		}
		if req.UserCode == "" {
			writeJSONError(w, http.StatusBadRequest, "user_code is required")
			return
		}
		if req.Action == "" {
			req.Action = "approve"
		}

		err := s.ApproveDeviceCode(req.UserCode, uuid.Nil)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "user_code": req.UserCode})
	}
}
