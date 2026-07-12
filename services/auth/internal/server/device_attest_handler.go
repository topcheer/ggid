package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// POST /api/v1/auth/devices/attest
func (h *Handler) handleDeviceAttest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return }
	var req struct {
		DeviceID      string `json:"device_id"`
		TPMQuote      string `json:"tpm_quote"`
		SecureBoot    bool   `json:"secure_boot"`
		CodeIntegrity bool   `json:"code_integrity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid JSON"); return }
	if req.DeviceID == "" { writeError(w, http.StatusBadRequest, "device_id required"); return }
	trustLevel := "none"
	if req.TPMQuote != "" && req.SecureBoot && req.CodeIntegrity {
		trustLevel = "full"
	} else if req.TPMQuote != "" || req.SecureBoot {
		trustLevel = "partial"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": req.DeviceID, "trust_level": trustLevel,
		"tpm_verified": req.TPMQuote != "", "secure_boot": req.SecureBoot,
		"code_integrity": req.CodeIntegrity, "attested_at": time.Now().UTC().Format(time.RFC3339),
		"mfa_override": trustLevel == "full",
	})
}
