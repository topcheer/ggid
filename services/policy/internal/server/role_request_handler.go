package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// roleRequest represents a self-service role access request.
type roleRequest struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RoleID       string    `json:"role_id"`
	TenantID     string    `json:"tenant_id"`
	Reason       string    `json:"reason"`
	Duration     string    `json:"duration,omitempty"` // e.g. "8h", "7d", "permanent"
	Status       string    `json:"status"`             // pending, approved, rejected, expired
	RequestedAt  string    `json:"requested_at"`
	ReviewedBy   string    `json:"reviewed_by,omitempty"`
	ReviewedAt   string    `json:"reviewed_at,omitempty"`
	ReviewComment string   `json:"review_comment,omitempty"`
	ApprovalChain []map[string]any `json:"approval_chain,omitempty"`
}

var roleRequestStore = struct {
	sync.RWMutex
	requests map[string]*roleRequest
}{requests: make(map[string]*roleRequest)}

// POST   /api/v1/policies/role-requests — create a self-service role request
// GET    /api/v1/policies/role-requests — list requests (filter by user_id, status, tenant_id)
// POST   /api/v1/policies/role-requests/{id}/approve — approve a request
// POST   /api/v1/policies/role-requests/{id}/reject — reject a request
func (s *HTTPServer) handleRoleRequests(w http.ResponseWriter, r *http.Request) {
	// Check for sub-paths: /{id}/approve or /{id}/reject
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/role-requests")
	path = strings.TrimPrefix(path, "/")

	if path != "" {
		parts := strings.SplitN(path, "/", 2)
		reqID := parts[0]
		if len(parts) == 2 && parts[1] == "approve" {
			s.approveRoleRequest(w, r, reqID)
			return
		}
		if len(parts) == 2 && parts[1] == "reject" {
			s.rejectRoleRequest(w, r, reqID)
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		s.createRoleRequest(w, r)
	case http.MethodGet:
		s.listRoleRequests(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) createRoleRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		RoleID   string `json:"role_id"`
		TenantID string `json:"tenant_id"`
		Reason   string `json:"reason"`
		Duration string `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" || req.RoleID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id and role_id are required")
		return
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	rr := &roleRequest{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		RoleID:      req.RoleID,
		TenantID:    req.TenantID,
		Reason:      req.Reason,
		Duration:    req.Duration,
		Status:      "pending",
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
		ApprovalChain: []map[string]any{
			{"step": 1, "approver_type": "manager", "status": "pending"},
			{"step": 2, "approver_type": "security_admin", "status": "pending"},
		},
	}

	roleRequestStore.Lock()
	roleRequestStore.requests[rr.ID] = rr
	roleRequestStore.Unlock()

	writeJSON(w, http.StatusCreated, rr)
}

func (s *HTTPServer) listRoleRequests(w http.ResponseWriter, r *http.Request) {
	userFilter := r.URL.Query().Get("user_id")
	statusFilter := r.URL.Query().Get("status")

	roleRequestStore.RLock()
	result := []*roleRequest{}
	for _, rr := range roleRequestStore.requests {
		if userFilter != "" && rr.UserID != userFilter {
			continue
		}
		if statusFilter != "" && rr.Status != statusFilter {
			continue
		}
		result = append(result, rr)
	}
	roleRequestStore.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"requests": result,
		"total":    len(result),
	})
}

func (s *HTTPServer) approveRoleRequest(w http.ResponseWriter, r *http.Request, reqID string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		ApproverID string `json:"approver_id"`
		Comment    string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ApproverID == "" {
		writeJSONError(w, http.StatusBadRequest, "approver_id is required")
		return
	}

	roleRequestStore.Lock()
	rr, exists := roleRequestStore.requests[reqID]
	if !exists {
		roleRequestStore.Unlock()
		writeJSONError(w, http.StatusNotFound, "role request not found")
		return
	}
	if rr.Status != "pending" {
		roleRequestStore.Unlock()
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("request already %s", rr.Status))
		return
	}
	rr.Status = "approved"
	rr.ReviewedBy = body.ApproverID
	rr.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	rr.ReviewComment = body.Comment
	// Update approval chain
	for i := range rr.ApprovalChain {
		rr.ApprovalChain[i]["status"] = "approved"
		rr.ApprovalChain[i]["approved_by"] = body.ApproverID
	}
	roleRequestStore.Unlock()

	writeJSON(w, http.StatusOK, rr)
}

func (s *HTTPServer) rejectRoleRequest(w http.ResponseWriter, r *http.Request, reqID string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		ApproverID string `json:"approver_id"`
		Comment    string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ApproverID == "" {
		writeJSONError(w, http.StatusBadRequest, "approver_id is required")
		return
	}

	roleRequestStore.Lock()
	rr, exists := roleRequestStore.requests[reqID]
	if !exists {
		roleRequestStore.Unlock()
		writeJSONError(w, http.StatusNotFound, "role request not found")
		return
	}
	if rr.Status != "pending" {
		roleRequestStore.Unlock()
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("request already %s", rr.Status))
		return
	}
	rr.Status = "rejected"
	rr.ReviewedBy = body.ApproverID
	rr.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	rr.ReviewComment = body.Comment
	for i := range rr.ApprovalChain {
		rr.ApprovalChain[i]["status"] = "rejected"
	}
	roleRequestStore.Unlock()

	writeJSON(w, http.StatusOK, rr)
}
