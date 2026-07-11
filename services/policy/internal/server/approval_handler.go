package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ApprovalRequest struct {
	ID             string        `json:"id"`
	RequestType    string        `json:"request_type"`
	Requester      string        `json:"requester"`
	ApproverChain  []string      `json:"approver_chain"`
	CurrentStep    int           `json:"current_step"`
	Status         string        `json:"status"` // pending, approved, rejected
	Payload        map[string]any `json:"payload,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	History        []ApprovalStep `json:"history"`
}

type ApprovalStep struct {
	Step      int       `json:"step"`
	Approver  string    `json:"approver"`
	Action    string    `json:"action"` // approved, rejected
	Timestamp time.Time `json:"timestamp"`
	Comment   string    `json:"comment,omitempty"`
}

var (
	approvalMu  sync.RWMutex
	approvals   = make(map[string]*ApprovalRequest)
)

// POST /api/v1/policies/approvals — create approval request
// GET /api/v1/policies/approvals/pending — list pending approvals
// POST /api/v1/policies/approvals/{id}/approve — approve current step
// POST /api/v1/policies/approvals/{id}/reject — reject
func (s *HTTPServer) handleApprovals(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// GET pending
	if strings.HasSuffix(path, "/pending") && r.Method == http.MethodGet {
		approvalMu.RLock()
		result := []*ApprovalRequest{}
		for _, a := range approvals {
			if a.Status == "pending" {
				result = append(result, a)
			}
		}
		approvalMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"approvals": result, "count": len(result)})
		return
	}

	// POST create
	if path == "/api/v1/policies/approvals" && r.Method == http.MethodPost {
		var req struct {
			RequestType   string   `json:"request_type"`
			Requester     string   `json:"requester"`
			ApproverChain []string `json:"approver_chain"`
			Payload       map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.RequestType == "" || req.Requester == "" || len(req.ApproverChain) == 0 {
			writeJSONError(w, http.StatusBadRequest, "request_type, requester, approver_chain required")
			return
		}
		now := time.Now().UTC()
		ar := &ApprovalRequest{
			ID: uuid.New().String(), RequestType: req.RequestType, Requester: req.Requester,
			ApproverChain: req.ApproverChain, CurrentStep: 0, Status: "pending",
			Payload: req.Payload, CreatedAt: now, UpdatedAt: now,
		}
		approvalMu.Lock()
		approvals[ar.ID] = ar
		approvalMu.Unlock()
		writeJSON(w, http.StatusCreated, ar)
		return
	}

	// POST approve/reject
	if (strings.HasSuffix(path, "/approve") || strings.HasSuffix(path, "/reject")) && r.Method == http.MethodPost {
		action := "approved"
		if strings.HasSuffix(path, "/reject") {
			action = "rejected"
		}
		// Extract ID: /api/v1/policies/approvals/{id}/approve
		parts := strings.Split(path, "/")
		if len(parts) < 6 {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		id := parts[5]

		approvalMu.Lock()
		ar, ok := approvals[id]
		if !ok {
			approvalMu.Unlock()
			writeJSONError(w, http.StatusNotFound, "approval request not found")
			return
		}
		if ar.Status != "pending" {
			approvalMu.Unlock()
			writeJSONError(w, http.StatusConflict, "approval already "+ar.Status)
			return
		}

		var req struct{ Comment string `json:"comment"` }
		_ = json.NewDecoder(r.Body).Decode(&req)

		step := ApprovalStep{
			Step: ar.CurrentStep, Approver: ar.ApproverChain[ar.CurrentStep],
			Action: action, Timestamp: time.Now().UTC(), Comment: req.Comment,
		}
		ar.History = append(ar.History, step)

		if action == "rejected" {
			ar.Status = "rejected"
		} else {
			ar.CurrentStep++
			if ar.CurrentStep >= len(ar.ApproverChain) {
				ar.Status = "approved"
			}
		}
		ar.UpdatedAt = time.Now().UTC()
		approvalMu.Unlock()

		writeJSON(w, http.StatusOK, ar)
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}
