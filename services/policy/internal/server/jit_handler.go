package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/repository"
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

	// SECURITY: Override body user_id with authenticated identity to prevent cross-user JIT submission.
	if uidStr := r.Header.Get("X-User-ID"); uidStr != "" {
		req.UserID = uidStr
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
	if !isAdminRequest(r) {
		writeJSONError(w, http.StatusForbidden, "admin scope required")
		return
	}
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
	// SECURITY: verify the JIT request belongs to the caller's tenant.
	if tc != nil && jitReq.TenantID != tc.TenantID {
		writeJSONError(w, http.StatusForbidden, "JIT request does not belong to caller's tenant")
		return
	}
	if jitReq.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "JIT request is not pending")
		return
	}

	// SECURITY: Prevent self-approval (separation of duties).
	if approverID == jitReq.UserID {
		writeJSONError(w, http.StatusForbidden, "cannot approve your own JIT request")
		return
	}

	expiresAt := time.Now().Add(time.Duration(jitReq.DurationMin) * time.Minute)

	// SECURITY: Prevent granting instance-level system roles via JIT.
	// Check BEFORE committing DB approval to avoid irreconcilable state.
	if s.roleSvc != nil {
		role, err := s.roleSvc.GetRole(r.Context(), jitReq.RoleID)
		if err != nil || role == nil {
			// SECURITY (R22 P1): fail-closed — if we can't resolve the role,
			// don't risk granting a system role.
			writeJSONError(w, http.StatusForbidden, "cannot resolve role for JIT approval")
			return
		}
		// Check SystemRole flag instead of hardcoded key list —
		// covers all system roles including custom keys.
		if role.SystemRole {
			writeJSONError(w, http.StatusForbidden, "cannot grant system role via JIT")
			return
		}
		// Also reject known privileged role keys as defense-in-depth.
		privilegedKeys := map[string]bool{
			"platform:admin": true, "system:admin": true, "tenant:admin": true, "administrator": true,
		}
		if privilegedKeys[role.Key] {
			writeJSONError(w, http.StatusForbidden, "cannot grant instance-level role via JIT")
			return
		}
	}

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
	if !isAdminRequest(r) {
		writeJSONError(w, http.StatusForbidden, "admin scope required")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	// SECURITY: verify the JIT request belongs to the caller's tenant.
	jitReq, err := s.jitRepo.GetByID(r.Context(), reqID)
	if err != nil || jitReq == nil {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
		return
	}
	if jitReq.TenantID != tc.TenantID {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
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
	if !isAdminRequest(r) {
		writeJSONError(w, http.StatusForbidden, "admin scope required")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Reason == "" {
		body.Reason = "manual_revoke"
	}

	// Fetch to get user/role for revocation.
	jitReq, err := s.jitRepo.GetByID(r.Context(), reqID)
	if err != nil || jitReq == nil {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
		return
	}
	// SECURITY: verify ownership
	if jitReq.TenantID != tc.TenantID {
		writeJSONError(w, http.StatusNotFound, "JIT request not found")
		return
	}

	if err := s.jitRepo.Revoke(r.Context(), reqID, body.Reason); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to revoke JIT request")
		return
	}

	// Revoke the role binding.
	if s.roleSvc != nil {
		_ = s.roleSvc.RevokeRoleTemporaryOnly(r.Context(), jitReq.UserID, jitReq.RoleID, domain.ScopeGlobal, tc.TenantID)
	}

	s.publishAuditEvent("jit.revoke", "success", "jit_request", reqID, tc.TenantID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// StartJITExpiryJanitor runs a background goroutine that periodically cleans up
// expired JIT requests and revokes their temporary role bindings.
// Call once at server startup: go StartJITExpiryJanitor(s, ctx)
func StartJITExpiryJanitor(s *HTTPServer, ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupExpiredJIT(s, ctx)
		}
	}
}

// cleanupExpiredJIT finds active JIT requests past their expiry, marks them expired,
// and revokes the corresponding role bindings.
func cleanupExpiredJIT(s *HTTPServer, ctx context.Context) {
	if s.jitRepo == nil {
		return
	}
	expired, err := s.jitRepo.ListExpired(ctx)
	if err != nil {
		log.Printf("JIT expiry cleanup: ListExpired error: %v", err)
		return
	}
	for _, jitReq := range expired {
		if err := s.jitRepo.MarkExpired(ctx, jitReq.ID); err != nil {
			log.Printf("JIT expiry cleanup: MarkExpired %s error: %v", jitReq.ID, err)
			continue
		}
		// Revoke the temporary role binding.
		if s.roleSvc != nil {
			_ = s.roleSvc.RevokeRoleTemporaryOnly(ctx, jitReq.UserID, jitReq.RoleID, domain.ScopeGlobal, jitReq.TenantID)
		}
		s.publishAuditEvent("jit.expired", "success", "jit_request", jitReq.ID, jitReq.TenantID)
		log.Printf("JIT expiry cleanup: expired request %s (user=%s role=%s)", jitReq.ID, jitReq.UserID, jitReq.RoleID)
	}
}
