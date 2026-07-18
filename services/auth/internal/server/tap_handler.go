package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/ggid/ggid/services/auth/internal/tap"
	"github.com/google/uuid"
)

// --- DTOs ---

type tapIssueRequest struct {
	UserID   string `json:"user_id"`
	Reason   string `json:"reason"`
	GroupID  string `json:"group_id"`
	TTLMin   int    `json:"ttl_minutes"`
}

type tapBatchRequest struct {
	UserIDs []string `json:"user_ids"`
	Reason  string   `json:"reason"`
	GroupID string   `json:"group_id"`
	TTLMin  int      `json:"ttl_minutes"`
}

type tapPolicyRequest struct {
	AllowedGroups []string `json:"allowed_groups"`
	MaxPerDay     int      `json:"max_per_day"`
	TTLMinutes    int      `json:"ttl_minutes"`
}

type tapBatchResponse struct {
	Issued []tapIssueResult   `json:"issued"`
	Errors []tapBatchError    `json:"errors,omitempty"`
}

type tapIssueResult struct {
	UserID string `json:"user_id"`
	TAPID  string `json:"tap_id"`
	Code   string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

type tapBatchError struct {
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}

// --- Setter ---

func (h *Handler) SetTAPEngine(engine *tap.Engine) {
	h.tapEngine = engine
}

func (h *Handler) SetTAPPolicyRepo(repo *repository.TAPPolicyRepository) {
	h.tapPolicyRepo = repo
}

// --- Handler ---

func (h *Handler) handleTAP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/tap")

	switch {
	case r.Method == http.MethodPost && path == "/batch":
		h.tapBatch(w, r)
	case r.Method == http.MethodGet && path == "/audit":
		h.tapAudit(w, r)
	case r.Method == http.MethodGet && path == "/policy":
		h.tapGetPolicy(w, r)
	case r.Method == http.MethodPut && path == "/policy":
		h.tapUpdatePolicy(w, r)
	case r.Method == http.MethodPost && (path == "" || path == "/"):
		h.tapIssue(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/user/"):
		h.tapListUser(w, r, strings.TrimPrefix(path, "/user/"))
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// --- Single Issue ---

func (h *Handler) tapIssue(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	var req tapIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.UserID == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	// Check group policy.
	if h.tapPolicyRepo != nil && req.GroupID != "" {
		if !h.tapPolicyRepo.IsGroupAllowed(r.Context(), tc.TenantID, req.GroupID) {
			errors.WriteSimpleAPIError(w, http.StatusForbidden, "POLICY_DENIED", "TAP not allowed for this group")
			return
		}
	}

	ttl := time.Duration(req.TTLMin) * time.Minute
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	if h.tapEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}

	code, rec, err := h.tapEngine.Issue(r.Context(), req.UserID, "admin", req.Reason, ttl)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to issue TAP")
		return
	}

	// Audit.
	h.publishAuditEventWithMeta(r,
		"tap.issue", "success",
		"temporary_access_pass", rec.ID, uuid.Nil,
		map[string]any{"user_id": req.UserID, "reason": req.Reason, "expires_at": rec.ExpiresAt},
	)

	writeJSON(w, http.StatusCreated, map[string]any{
		"tap_id":     rec.ID,
		"code":       code,
		"user_id":    req.UserID,
		"expires_at": rec.ExpiresAt,
	})
}

// --- Batch Issue ---

func (h *Handler) tapBatch(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	var req tapBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if len(req.UserIDs) == 0 {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_ids array is required")
		return
	}
	if len(req.UserIDs) > 100 {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "max 100 users per batch")
		return
	}

	// Check group policy.
	if h.tapPolicyRepo != nil && req.GroupID != "" {
		if !h.tapPolicyRepo.IsGroupAllowed(r.Context(), tc.TenantID, req.GroupID) {
			errors.WriteSimpleAPIError(w, http.StatusForbidden, "POLICY_DENIED", "TAP not allowed for this group")
			return
		}
	}

	ttl := time.Duration(req.TTLMin) * time.Minute
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	if h.tapEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}

	resp := tapBatchResponse{}
	for _, uid := range req.UserIDs {
		code, rec, err := h.tapEngine.Issue(r.Context(), uid, "admin", req.Reason, ttl)
		if err != nil {
			resp.Errors = append(resp.Errors, tapBatchError{UserID: uid, Error: err.Error()})
			continue
		}
		resp.Issued = append(resp.Issued, tapIssueResult{
			UserID:    uid,
			TAPID:     rec.ID,
			Code:      code,
			ExpiresAt: rec.ExpiresAt.Format(time.RFC3339),
		})
	}

	// Audit batch.
	h.publishAuditEventWithMeta(r,
		"tap.batch_issue", "success",
		"temporary_access_pass", "", uuid.Nil,
		map[string]any{"count": len(resp.Issued), "reason": req.Reason},
	)

	writeJSON(w, http.StatusOK, resp)
}

// --- Audit Log ---

func (h *Handler) tapAudit(w http.ResponseWriter, r *http.Request) {
	_, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"message": "specify user_id query param for TAP audit trail"})
		return
	}

	if h.tapEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	records, err := h.tapEngine.ListUserTAPs(r.Context(), userID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to retrieve TAP audit")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// --- Policy ---

func (h *Handler) tapGetPolicy(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if h.tapPolicyRepo == nil {
		writeJSON(w, http.StatusOK, &repository.TAPPolicy{
			TenantID:      tc.TenantID,
			AllowedGroups: []string{},
			MaxPerDay:     10,
			TTLMinutes:    15,
		})
		return
	}

	policy, err := h.tapPolicyRepo.Get(r.Context(), tc.TenantID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to get policy")
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) tapUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	var req tapPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.MaxPerDay <= 0 {
		req.MaxPerDay = 10
	}
	if req.TTLMinutes <= 0 {
		req.TTLMinutes = 15
	}

	policy := &repository.TAPPolicy{
		TenantID:      tc.TenantID,
		AllowedGroups: req.AllowedGroups,
		MaxPerDay:     req.MaxPerDay,
		TTLMinutes:    req.TTLMinutes,
	}

	if h.tapPolicyRepo != nil {
		if err := h.tapPolicyRepo.Upsert(r.Context(), policy); err != nil {
			errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to update policy")
			return
		}
	}

	// Audit.
	h.publishAuditEventWithMeta(r,
		"tap.policy.update", "success",
		"tap_policy", "", uuid.Nil,
		map[string]any{"allowed_groups": req.AllowedGroups, "max_per_day": req.MaxPerDay},
	)

	writeJSON(w, http.StatusOK, policy)
}

// --- List User TAPs ---

func (h *Handler) tapListUser(w http.ResponseWriter, r *http.Request, userID string) {
	_, err := tenant.FromContext(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required")
		return
	}

	if h.tapEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	records, err := h.tapEngine.ListUserTAPs(r.Context(), userID)
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to list TAPs")
		return
	}
	writeJSON(w, http.StatusOK, records)
}
