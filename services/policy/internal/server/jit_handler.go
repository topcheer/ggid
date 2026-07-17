package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/repository"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// SetJITRepo injects the JIT request repository.
func (s *HTTPServer) SetJITRepo(repo *repository.JITRequestRepository) {
	s.jitRepo = repo
}

// handleJIT routes JIT elevation endpoints.
// POST   /api/v1/policies/jit/request              — submit request
// GET    /api/v1/policies/jit/requests              — list (filter by status/user_id)
// GET    /api/v1/policies/jit/active                — active elevations
// POST   /api/v1/policies/jit/requests/{id}/approve  — approve
// POST   /api/v1/policies/jit/requests/{id}/reject   — reject
// POST   /api/v1/policies/jit/requests/{id}/revoke   — revoke
func (s *HTTPServer) handleJIT(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/api/v1/policies/jit/request" && r.Method == http.MethodPost {
		s.jitCreateRequest(w, r)
		return
	}
	if path == "/api/v1/policies/jit/requests" && r.Method == http.MethodGet {
		s.jitListRequests(w, r)
		return
	}
	if path == "/api/v1/policies/jit/active" && r.Method == http.MethodGet {
		s.jitListActive(w, r)
		return
	}

	// Sub-path routing: /api/v1/policies/jit/requests/{id}/{action}
	if strings.HasPrefix(path, "/api/v1/policies/jit/requests/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/policies/jit/requests/"), "/")
		if len(parts) == 2 {
			reqID, err := uuid.Parse(parts[0])
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid request id")
				return
			}
			switch parts[1] {
			case "approve":
				s.jitApprove(w, r, reqID)
			case "reject":
				s.jitReject(w, r, reqID)
			case "revoke":
				s.jitRevoke(w, r, reqID)
			default:
				writeJSONError(w, http.StatusNotFound, "not found")
			}
			return
		}
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) jitCreateRequest(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req struct {
		UserID      string `json:"user_id"`
		RoleID      string `json:"role_id"`
		Reason      string `json:"reason"`
		DurationMin int    `json:"duration_min"`
		ScopeType   string `json:"scope_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" || req.RoleID == "" || req.Reason == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id, role_id, and reason are required")
		return
	}
	if req.DurationMin <= 0 || req.DurationMin > 480 {
		req.DurationMin = 60 // default 1h, max 8h
	}
	if req.ScopeType == "" {
		req.ScopeType = "tenant"
	}

	userID, _ := uuid.Parse(req.UserID)
	roleID, _ := uuid.Parse(req.RoleID)

	jitReq := &repository.JITRequest{
		ID:          uuid.New(),
		TenantID:    tc.TenantID,
		UserID:      userID,
		RoleID:      roleID,
		ScopeType:   req.ScopeType,
		Reason:      req.Reason,
		DurationMin: req.DurationMin,
		Status:      "pending",
	}

	if err := s.jitRepo.Create(r.Context(), jitReq); err != nil {
		log.Printf("JIT create error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create JIT request")
		return
	}

	s.publishAuditEvent("jit.request", "success", "jit_request", jitReq.ID, tc.TenantID)

	writeJSON(w, http.StatusCreated, jitReq)
}

func (s *HTTPServer) jitListRequests(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	status := r.URL.Query().Get("status")
	var userID *uuid.UUID
	if uidStr := r.URL.Query().Get("user_id"); uidStr != "" {
		if uid, err := uuid.Parse(uidStr); err == nil {
			userID = &uid
		}
	}

	requests, err := s.jitRepo.List(r.Context(), tc.TenantID, status, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list JIT requests")
		return
	}
	if requests == nil {
		requests = []*repository.JITRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests, "total": len(requests)})
}

func (s *HTTPServer) jitListActive(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	requests, err := s.jitRepo.List(r.Context(), tc.TenantID, "active", nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list active JIT requests")
		return
	}
	if requests == nil {
		requests = []*repository.JITRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"active": requests, "total": len(requests)})
}

func (s *HTTPServer) jitApprove(w http.ResponseWriter, r *http.Request, reqID uuid.UUID) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	// Get approver ID from header.
	var approverID uuid.UUID
	if uidStr := r.Header.Get("X-User-ID"); uidStr != "" {
		approverID, _ = uuid.Parse(uidStr)
	}

	// Fetch the request to get duration.
	jitReq, err := s.jitRepo.GetByID(r.Context(), reqID)
	if err != nil || jitReq == nil {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
		return
	}
	if jitReq.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "JIT request is not pending")
		return
	}

	expiresAt := time.Now().Add(time.Duration(jitReq.DurationMin) * time.Minute)

	if err := s.jitRepo.Approve(r.Context(), reqID, approverID, expiresAt); err != nil {
		log.Printf("JIT approve error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to approve JIT request")
		return
	}

	// Bind the role temporarily (expires_at set via UserRole).
	if s.roleSvc != nil {
		expiresAt := time.Now().Add(time.Duration(jitReq.DurationMin) * time.Minute)
		if err := s.roleSvc.AssignRole(r.Context(), jitReq.UserID, jitReq.RoleID, domain.ScopeGlobal, tc.TenantID, approverID, &expiresAt); err != nil {
			log.Printf("JIT approve: AssignRole failed: %v", err)
		}
	}

	s.publishAuditEvent("jit.approve", "success", "jit_request", reqID, tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "active", "expires_at": expiresAt})
}

func (s *HTTPServer) jitReject(w http.ResponseWriter, r *http.Request, reqID uuid.UUID) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var approverID uuid.UUID
	if uidStr := r.Header.Get("X-User-ID"); uidStr != "" {
		approverID, _ = uuid.Parse(uidStr)
	}

	if err := s.jitRepo.Reject(r.Context(), reqID, approverID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to reject JIT request")
		return
	}

	s.publishAuditEvent("jit.reject", "success", "jit_request", reqID, tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *HTTPServer) jitRevoke(w http.ResponseWriter, r *http.Request, reqID uuid.UUID) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		body.Reason = "manual_revoke"
	}

	// Fetch to get user/role for revocation.
	jitReq, err := s.jitRepo.GetByID(r.Context(), reqID)
	if err != nil || jitReq == nil {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
		return
	}

	if err := s.jitRepo.Revoke(r.Context(), reqID, body.Reason); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to revoke JIT request")
		return
	}

	// Revoke the role binding.
	if s.roleSvc != nil {
		_ = s.roleSvc.RevokeRole(r.Context(), jitReq.UserID, jitReq.RoleID, domain.ScopeGlobal, tc.TenantID)
	}

	s.publishAuditEvent("jit.revoke", "success", "jit_request", reqID, tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
