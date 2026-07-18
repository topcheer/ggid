package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// loginFlowStep represents one step in a login flow.
type loginFlowStep struct {
	Step       string `json:"step"` // password, mfa_totp, mfa_sms, saml, oidc, biometric
	Method     string `json:"method"`
	Success    bool   `json:"success"`
	DurationMs int    `json:"duration_ms"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	Timestamp  string `json:"timestamp"`
}

// loginFlowRecord stores a complete login flow recording.
type loginFlowRecord struct {
	ID         string           `json:"id"`
	UserID     string           `json:"user_id"`
	TenantID   string           `json:"tenant_id"`
	Steps      []loginFlowStep  `json:"steps"`
	Outcome    string           `json:"outcome"` // success, failed, abandoned
	TotalDurationMs int         `json:"total_duration_ms"`
	RecordedAt string           `json:"recorded_at"`
}

var loginFlowStore = struct {
	sync.RWMutex
	records []loginFlowRecord
}{records: []loginFlowRecord{}}

// POST /api/v1/auth/login-flow/record
func (h *Handler) handleLoginFlowRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID   string          `json:"user_id"`
		TenantID string          `json:"tenant_id"`
		Steps    []loginFlowStep `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if len(req.Steps) == 0 {
		writeJSONError(w, http.StatusBadRequest, "steps must not be empty")
		return
	}

	totalDuration := 0
	allSuccess := true
	for _, s := range req.Steps {
		totalDuration += s.DurationMs
		if !s.Success {
			allSuccess = false
		}
	}

	outcome := "success"
	if !allSuccess {
		outcome = "failed"
	}

	record := loginFlowRecord{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		TenantID:        req.TenantID,
		Steps:           req.Steps,
		Outcome:         outcome,
		TotalDurationMs: totalDuration,
		RecordedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	loginFlowStore.Lock()
	loginFlowStore.records = append(loginFlowStore.records, record)
	if len(loginFlowStore.records) > 1000 {
		loginFlowStore.records = loginFlowStore.records[len(loginFlowStore.records)-1000:]
	}
	loginFlowStore.Unlock()

	writeJSON(w, http.StatusCreated, record)
}
