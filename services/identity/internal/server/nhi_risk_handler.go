package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// nhiRiskScanRequest is the DTO for triggering a manual risk scan.
type nhiRiskScanRequest struct {
	NHIID         string  `json:"nhi_id"`
	Endpoint      string  `json:"endpoint"`
	CallsPerHour  float64 `json:"calls_per_hour"`
	IP            string  `json:"ip"`
	Hour          int     `json:"hour"` // 0-23, -1 = use current hour
}

func (h *HTTPHandler) handleNHIRisk(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/nhi")

	switch {
	case strings.HasPrefix(path, "/risk-alerts") && r.Method == http.MethodGet:
		h.nhiRiskAlerts(w, r)
	case strings.HasPrefix(path, "/risk/scan") && r.Method == http.MethodPost:
		h.nhiRiskScan(w, r)
	case strings.Contains(path, "/risk") && r.Method == http.MethodGet:
		// GET /api/v1/identity/nhi/:id/risk
		h.nhiGetRisk(w, r, path)
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// nhiGetRisk returns the risk score for a specific NHI.
// Path format: /<nhi_id>/risk
func (h *HTTPHandler) nhiGetRisk(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_PATH", "path format: /<nhi_id>/risk")
		return
	}

	nhiID, err := uuid.Parse(parts[0])
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid NHI ID")
		return
	}

	if h.nhiRiskEngine == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"nhi_id": nhiID,
			"score": 0,
			"level": "unknown",
			"message": "NHI risk engine not configured",
		})
		return
	}

	score := h.nhiRiskEngine.GetRiskScore(nhiID)
	if score == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"nhi_id": nhiID,
			"score": 0,
			"level": "unknown",
			"message": "no risk evaluation performed yet",
		})
		return
	}

	writeJSON(w, http.StatusOK, score)
}

// nhiRiskAlerts returns all high-risk NHIs.
func (h *HTTPHandler) nhiRiskAlerts(w http.ResponseWriter, r *http.Request) {
	if h.nhiRiskEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	threshold := 50 // high and above
	alerts := h.nhiRiskEngine.ListHighRisk(threshold)
	if alerts == nil {
		alerts = []*NHIRiskScore{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alerts":    alerts,
		"count":     len(alerts),
		"threshold": threshold,
	})
}

// nhiRiskScan triggers a manual risk evaluation for an NHI.
func (h *HTTPHandler) nhiRiskScan(w http.ResponseWriter, r *http.Request) {
	var req nhiRiskScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.NHIID == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "nhi_id is required")
		return
	}

	nhiID, err := uuid.Parse(req.NHIID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_ID", "invalid NHI ID format")
		return
	}

	if h.nhiRiskEngine == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"nhi_id":  nhiID,
			"score":   0,
			"level":   "unknown",
			"message": "NHI risk engine not configured",
		})
		return
	}

	hour := req.Hour
	if hour < 0 || hour > 23 {
		hour = time.Now().Hour()
	}

	activity := CurrentActivity{
		NHIID:        req.NHIID,
		Endpoint:     req.Endpoint,
		CallsPerHour: req.CallsPerHour,
		IP:           req.IP,
		Hour:         hour,
	}

	score := h.nhiRiskEngine.EvaluateRisk(nhiID, activity)

	// High-risk auto-trigger: SOAR playbook (revoke + notify).
	if score.Score >= 70 {
		score.Signals["soar_triggered"] = true
		score.Signals["soar_action"] = "token_revoke + admin_notify"
	}

	writeJSON(w, http.StatusOK, score)
}
