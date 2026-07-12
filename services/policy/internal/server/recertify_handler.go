package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// recertificationDecision represents a single role recertification decision.
type recertificationDecision struct {
	RoleID   string `json:"role_id"`
	UserID   string `json:"user_id"`
	Action   string `json:"action"` // keep, remove, modify
	NewRole  string `json:"new_role_id,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// recertificationRecord stores a batch recertification submission.
type recertificationRecord struct {
	ID          string                   `json:"id"`
	ManagerID   string                   `json:"manager_id"`
	TenantID    string                   `json:"tenant_id"`
	SubmittedAt string                   `json:"submitted_at"`
	Decisions   []recertificationDecision `json:"decisions"`
	Status      string                   `json:"status"` // pending, processed
	NotificationsSent int                `json:"notifications_sent"`
}

var recertStore = struct {
	sync.RWMutex
	records map[string]*recertificationRecord
}{records: make(map[string]*recertificationRecord)}

// POST /api/v1/policies/recertify
// Body: {"manager_id": "...", "tenant_id": "...", "decisions": [...]}
// Manager submits access recertification decisions for subordinates.
func (s *HTTPServer) handleRecertify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ManagerID  string                    `json:"manager_id"`
		TenantID   string                    `json:"tenant_id"`
		Decisions  []recertificationDecision `json:"decisions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.ManagerID == "" {
		writeJSONError(w, http.StatusBadRequest, "manager_id is required")
		return
	}
	if _, err := uuid.Parse(req.ManagerID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid manager_id")
		return
	}
	if len(req.Decisions) == 0 {
		writeJSONError(w, http.StatusBadRequest, "decisions must not be empty")
		return
	}

	// Validate actions
	validActions := map[string]bool{"keep": true, "remove": true, "modify": true}
	for i, d := range req.Decisions {
		if d.RoleID == "" || d.UserID == "" {
			writeJSONError(w, http.StatusBadRequest, "each decision requires role_id and user_id")
			return
		}
		if !validActions[d.Action] {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decision %d: action must be keep, remove, or modify", i+1))
			return
		}
		if d.Action == "modify" && d.NewRole == "" {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decision %d: new_role_id required for modify action", i+1))
			return
		}
	}

	recID := uuid.New().String()
	rec := &recertificationRecord{
		ID:          recID,
		ManagerID:   req.ManagerID,
		TenantID:    req.TenantID,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Decisions:   req.Decisions,
		Status:      "pending",
	}

	// Process decisions: simulate notifications
	notificationsSent := 0
	uniqueUsers := map[string]bool{}
	for _, d := range req.Decisions {
		if !uniqueUsers[d.UserID] {
			uniqueUsers[d.UserID] = true
			notificationsSent++
		}
	}
	rec.NotificationsSent = notificationsSent
	rec.Status = "processed"

	recertStore.Lock()
	recertStore.records[recID] = rec
	recertStore.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":                 rec.ID,
		"status":             rec.Status,
		"manager_id":         rec.ManagerID,
		"total_decisions":    len(rec.Decisions),
		"notifications_sent": rec.NotificationsSent,
		"submitted_at":       rec.SubmittedAt,
	})
}
