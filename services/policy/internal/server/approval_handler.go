package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

type ApprovalRequest struct {
	ID            string         `json:"id"`
	RequestType   string         `json:"request_type"`
	Requester     string         `json:"requester"`
	ApproverChain []string       `json:"approver_chain"`
	CurrentStep   int            `json:"current_step"`
	Status        string         `json:"status"`
	Payload       map[string]any `json:"payload"`
	History       []ApprovalStep `json:"history"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type ApprovalStep struct {
	Step      int       `json:"step"`
	Approver  string    `json:"approver"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Comment   string    `json:"comment"`
}

func (s *HTTPServer) handleApprovals(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// SECURITY: Extract caller tenant for BOLA protection.
	callerTenant := ""
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil && tc.TenantID != uuid.Nil {
		callerTenant = tc.TenantID.String()
	}

	if strings.HasSuffix(path, "/pending") && r.Method == http.MethodGet {
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_approvals")
			for _, row := range rows {
				if pmGetString(row, "status") == "pending" && pmGetString(row, "tenant_id") == callerTenant {
					result = append(result, row)
				}
			}
		}
		if result == nil {
			result = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"approvals": result, "count": len(result)})
		return
	}

	if path == "/api/v1/policies/approvals" && r.Method == http.MethodPost {
		var req struct {
			RequestType   string         `json:"request_type"`
			Requester     string         `json:"requester"`
			ApproverChain []string       `json:"approver_chain"`
			Payload       map[string]any `json:"payload"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
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
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_approvals", ar.ID, map[string]any{
				"tenant_id": callerTenant, "request_type": ar.RequestType, "requester": ar.Requester,
				"approver_chain": ar.ApproverChain, "current_step": ar.CurrentStep,
				"status": ar.Status, "payload": ar.Payload,
			})
		}
		writeJSON(w, http.StatusCreated, ar)
		return
	}

	if (strings.HasSuffix(path, "/approve") || strings.HasSuffix(path, "/reject")) && r.Method == http.MethodPost {
		action := "approved"
		if strings.HasSuffix(path, "/reject") {
			action = "rejected"
		}
		parts := strings.Split(path, "/")
		if len(parts) < 6 {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		id := parts[5]
		if s.policyMap != nil {
			row, _ := s.policyMap.Get(r.Context(), "policy_approvals", id)
			if row == nil {
				writeJSONError(w, http.StatusNotFound, "approval request not found")
				return
			}
			// SECURITY: Verify approval belongs to caller's tenant.
			if pmGetString(row, "tenant_id") != callerTenant {
				writeJSONError(w, http.StatusNotFound, "approval request not found")
				return
			}
			row["status"] = action
			s.policyMap.Store(r.Context(), "policy_approvals", id, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": action, "id": id})
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}
