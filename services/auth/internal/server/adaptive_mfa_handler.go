package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// adaptiveMFAStore tracks recent MFA decisions for caching.
var (
	adaptiveMFAMu sync.RWMutex
	adaptiveDecisions = make(map[string]*adaptiveMFAResult)
)

type adaptiveMFAResult struct {
	Required   bool   `json:"required"`
	FactorType string `json:"factor_type"`
	Reason     string `json:"reason"`
	Score      int    `json:"risk_score"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// POST /api/v1/auth/adaptive-mfa/evaluate
// Body: {"risk_score": 45, "user_id": "...", "device_trust": "trusted", "has_mfa": true}
func (h *Handler) handleAdaptiveMFA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		RiskScore   int    `json:"risk_score"`
		UserID      string `json:"user_id"`
		DeviceTrust string `json:"device_trust"`
		HasMFA      bool   `json:"has_mfa"`
		NewDevice   bool   `json:"new_device"`
		NewLocation bool   `json:"new_location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Decision matrix
	var result adaptiveMFAResult
	result.Score = req.RiskScore

	switch {
	case req.RiskScore >= 70:
		result.Required = true
		result.FactorType = "webauthn"
		result.Reason = "critical_risk_score"
	case req.RiskScore >= 40:
		result.Required = true
		result.FactorType = "totp"
		result.Reason = "high_risk_score"
	case req.NewDevice:
		result.Required = true
		result.FactorType = "totp"
		result.Reason = "new_device"
	case req.NewLocation:
		result.Required = true
		result.FactorType = "sms"
		result.Reason = "new_location"
	case req.DeviceTrust == "untrusted":
		result.Required = true
		result.FactorType = "totp"
		result.Reason = "untrusted_device"
	default:
		result.Required = false
		result.FactorType = ""
		result.Reason = "low_risk_trusted_context"
	}

	if !req.HasMFA && result.Required {
		result.FactorType = "totp"
		result.Reason += "_no_webauthn"
	}

	result.EvaluatedAt = time.Now().UTC()

	adaptiveMFAMu.Lock()
	adaptiveDecisions[req.UserID] = &result

	// PG write-through
	if h.memMapRepo != nil {
		data, _ := json.Marshal(result)
		var m map[string]any
		json.Unmarshal(data, &m)
		h.memMapRepo.StoreJSON(r.Context(), "auth_adaptive_mfa_decisions", req.UserID, m)
	}
	adaptiveMFAMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"required":     result.Required,
		"factor_type":  result.FactorType,
		"reason":       result.Reason,
		"risk_score":   result.Score,
		"user_id":      req.UserID,
		"evaluated_at": result.EvaluatedAt.Format(time.RFC3339),
	})
}
