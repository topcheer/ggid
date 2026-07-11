package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// tenantFromContext extracts the tenant UUID from the request context.
func tenantFromContext(ctx context.Context) uuid.UUID {
	tc, err := ggidtenant.FromContext(ctx)
	if err != nil || tc == nil {
		return uuid.Nil
	}
	return tc.TenantID
}

// handleAccessRequests routes access request endpoints.
// POST   /api/v1/access-requests              — create request
// GET    /api/v1/access-requests               — list requests (optional ?status=pending)
// POST   /api/v1/access-requests/{id}/approve  — approve
// POST   /api/v1/access-requests/{id}/deny     — deny
func (h *HTTPHandler) handleAccessRequests(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case path == "/api/v1/access-requests" && r.Method == http.MethodPost:
		h.createAccessRequest(w, r)
	case path == "/api/v1/access-requests" && r.Method == http.MethodGet:
		h.listAccessRequests(w, r)
	case strings.HasSuffix(path, "/approve") && r.Method == http.MethodPost:
		h.approveAccessRequest(w, r)
	case strings.HasSuffix(path, "/deny") && r.Method == http.MethodPost:
		h.denyAccessRequest(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) createAccessRequest(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	var body struct {
		RequesterID  string `json:"requester_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	requesterID, err := uuid.Parse(body.RequesterID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid requester_id")
		return
	}

	req, err := h.accessRequestSvc.CreateAccessRequest(
		ctx, tenantFromContext(ctx), requesterID,
		domain.ResourceType(body.ResourceType), body.ResourceID, body.Reason,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, req)
}

func (h *HTTPHandler) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	status := domain.AccessRequestStatus(r.URL.Query().Get("status"))

	requests, err := h.accessRequestSvc.ListRequests(ctx, tenantFromContext(ctx), status)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"requests": requests,
		"count":    len(requests),
	})
}

func (h *HTTPHandler) approveAccessRequest(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		writeError(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	requestID, err := uuid.Parse(parts[4])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request ID")
		return
	}

	var body struct {
		ApproverID string `json:"approver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	approverID, err := uuid.Parse(body.ApproverID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid approver_id")
		return
	}

	req, err := h.accessRequestSvc.ApproveAccessRequest(ctx, requestID, approverID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, req)
}

func (h *HTTPHandler) denyAccessRequest(w http.ResponseWriter, r *http.Request) {
	ctx, ok := injectTenant(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID header")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		writeError(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	requestID, err := uuid.Parse(parts[4])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request ID")
		return
	}

	var body struct {
		ApproverID    string `json:"approver_id"`
		DenialReason  string `json:"denial_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	approverID, err := uuid.Parse(body.ApproverID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid approver_id")
		return
	}

	req, err := h.accessRequestSvc.DenyAccessRequest(ctx, requestID, approverID, body.DenialReason)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, req)
}
