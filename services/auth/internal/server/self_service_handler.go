package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// handleSelfServiceDevices provides user-scoped device management.
// GET    /api/v1/self-service/devices          — list own devices (fingerprints + MFA)
// DELETE /api/v1/self-service/devices/{id}      — revoke/trust-remove a device
func (h *Handler) handleSelfServiceDevices(w http.ResponseWriter, r *http.Request) {
	// Authenticate via JWT
	claims, err := h.parseTokenFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	userID, err := userIDFromClaims(claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	tenantID, err := tenantIDFromClaimsOrHeader(claims, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid tenant_id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listSelfServiceDevices(w, r, userID, tenantID)
	case http.MethodDelete:
		h.revokeSelfServiceDevice(w, r, userID, tenantID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) listSelfServiceDevices(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tenantID uuid.UUID) {
	devices := make([]map[string]any, 0)

	// 1. Gather device fingerprints from DB (memMapRepo)
	if h.memMapRepo != nil {
		rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_device_fingerprints_json")
		for _, row := range rows {
			if uid, _ := row["user_id"].(string); uid == userID.String() {
				devices = append(devices, row)
			}
		}
	}

	// 2. Gather MFA devices (authenticator, passkey, etc.)
	mfaSvc := h.authSvc.MFAService()
	if mfaSvc != nil {
		mfaDevices, _ := mfaSvc.ListDevices(r.Context(), userID)
		for _, d := range mfaDevices {
			devices = append(devices, map[string]any{
				"id":         d.ID,
				"type":       "mfa",
				"name":       d.Name,
				"algorithm":  d.Algorithm,
				"created_at": d.CreatedAt,
			})
		}
	}

	// 3. Gather trusted devices
	if h.memMapRepo != nil {
		rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_trusted_devices_json")
		for _, row := range rows {
			if uid, _ := row["user_id"].(string); uid == userID.String() {
				row["trusted"] = true
				devices = append(devices, row)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"devices":      devices,
		"device_count": len(devices),
		"user_id":      userID.String(),
	})
}

func (h *Handler) revokeSelfServiceDevice(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tenantID uuid.UUID) {
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/v1/self-service/devices/")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

// revoked holds whether any device was successfully removed.
	revoked := false

	// Try to remove from device fingerprints (DB) — verify ownership + tenant
	if h.memMapRepo != nil {
		rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_device_fingerprints_json")
		for _, row := range rows {
			if id, _ := row["id"].(string); id == deviceID {
				if uid, _ := row["user_id"].(string); uid == userID.String() {
					// KB-278: verify tenant ownership before deleting
					if tid, _ := row["tenant_id"].(string); tid != "" && tid != tenantID.String() {
						writeError(w, http.StatusForbidden, "cross-tenant device access denied")
						return
					}
					if err := h.memMapRepo.DeleteJSON(r.Context(), "auth_device_fingerprints_json", deviceID); err == nil {
						revoked = true
					}
				}
				break
			}
		}
	}

	// Try to remove trusted device — verify ownership + tenant
	if h.memMapRepo != nil && !revoked {
		rows, _ := h.memMapRepo.ListJSON(r.Context(), "auth_trusted_devices_json")
		for _, row := range rows {
			if id, _ := row["id"].(string); id == deviceID {
				if uid, _ := row["user_id"].(string); uid == userID.String() {
					// KB-278: verify tenant ownership
					if tid, _ := row["tenant_id"].(string); tid != "" && tid != tenantID.String() {
						writeError(w, http.StatusForbidden, "cross-tenant device access denied")
						return
					}
					if err := h.memMapRepo.DeleteJSON(r.Context(), "auth_trusted_devices_json", deviceID); err == nil {
						revoked = true
					}
				}
				break
			}
		}
	}

	// Try to disable MFA device
	if !revoked {
		mfaSvc := h.authSvc.MFAService()
		if mfaSvc != nil {
			parsedDevID, parseErr := uuid.Parse(deviceID)
			if parseErr == nil {
				devices, _ := mfaSvc.ListDevices(r.Context(), userID)
				for _, d := range devices {
					if d.ID == parsedDevID {
						if err := mfaSvc.DisableMFA(r.Context(), parsedDevID); err == nil {
							revoked = true
						}
						break
					}
				}
			}
		}
	}

	if !revoked {
		writeError(w, http.StatusNotFound, "device not found or not owned by user")
		return
	}

	// Audit
	h.publishAuditEvent("self_service.device.revoke", "success", tenantID, userID)

	writeJSON(w, http.StatusOK, map[string]any{
		"revoked":   true,
		"device_id": deviceID,
		"timestamp": time.Now().UTC(),
	})
}

// handleSelfServiceSessions provides user-scoped session management.
// GET /api/v1/self-service/sessions — list own active sessions
func (h *Handler) handleSelfServiceSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.parseTokenFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	userID, err := userIDFromClaims(claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	tenantID, err := tenantIDFromClaimsOrHeader(claims, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid tenant_id required")
		return
	}

	sessions, err := h.authSvc.ListSessions(r.Context(), tenantID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
		"count":    len(sessions),
		"user_id":  userID.String(),
	})
}

// handleMFASelfRemove lets a user remove their own MFA factor after re-auth.
// DELETE /api/v1/self-service/mfa/{factor_id}
// Requires step-up verification token in request.
func (h *Handler) handleMFASelfRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.parseTokenFromHeader(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	userID, err := userIDFromClaims(claims)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	tenantID, err := tenantIDFromClaimsOrHeader(claims, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid tenant_id required")
		return
	}

	factorID := strings.TrimPrefix(r.URL.Path, "/api/v1/self-service/mfa/")
	if factorID == "" {
		writeError(w, http.StatusBadRequest, "factor_id is required in path")
		return
	}

	// Require re-auth: step-up token
	var req struct {
		StepUpToken string `json:"step_up_token"`
	}
	// Body is optional (token may be in query param)
	json.NewDecoder(r.Body).Decode(&req)
	if req.StepUpToken == "" {
		req.StepUpToken = r.URL.Query().Get("step_up_token")
	}
	if req.StepUpToken == "" {
		writeError(w, http.StatusForbidden, "step-up verification required for MFA removal")
		return
	}
	if err := h.authSvc.ValidateStepUpToken(r.Context(), req.StepUpToken, userID); err != nil {
		writeError(w, http.StatusForbidden, "invalid or expired step-up token")
		return
	}

	// Verify the MFA factor belongs to this user
	mfaSvc := h.authSvc.MFAService()
	if mfaSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "MFA service not available")
		return
	}

	parsedFactorID, parseErr := uuid.Parse(factorID)
	if parseErr != nil {
		writeError(w, http.StatusBadRequest, "invalid factor_id format")
		return
	}

	devices, _ := mfaSvc.ListDevices(r.Context(), userID)
	owned := false
	for _, d := range devices {
		if d.ID == parsedFactorID {
			owned = true
			break
		}
	}
	if !owned {
		writeError(w, http.StatusNotFound, "MFA factor not found or not owned by user")
		return
	}

	if err := mfaSvc.DisableMFA(r.Context(), parsedFactorID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove MFA factor")
		return
	}

	h.publishAuditEvent("self_service.mfa.remove", "success", tenantID, userID)

	writeJSON(w, http.StatusOK, map[string]any{
		"removed":    true,
		"factor_id":  factorID,
		"user_id":    userID.String(),
		"timestamp":  time.Now().UTC(),
	})
}
