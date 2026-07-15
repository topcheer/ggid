package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AccessRequest represents a user's request for elevated access.
type AccessRequest struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	RequesterID  string    `json:"requester_id"`
	RoleID       string    `json:"role_id"`
	Justification string   `json:"justification"`
	Status       string    `json:"status"` // pending, approved, rejected, expired
	ApproverID   string    `json:"approver_id,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote   string    `json:"review_note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// accessRequestStore holds access requests in memory.
type accessRequestStore struct {
	mu       sync.RWMutex
	requests map[string]*AccessRequest
}

var accessRequests = &accessRequestStore{requests: make(map[string]*AccessRequest)}

// POST /api/v1/policies/access-requests          — create access request
// GET  /api/v1/policies/access-requests           — list (filter by status/requester)
// GET  /api/v1/policies/access-requests/pending   — list pending requests
// POST /api/v1/policies/access-requests/{id}/approve — approve request
// POST /api/v1/policies/access-requests/{id}/reject  — reject request
func (s *HTTPServer) handleAccessRequests(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/access-requests")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			s.createAccessRequest(w, r)
		case http.MethodGet:
			s.listAccessRequests(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Sub-paths
	parts := strings.Split(strings.Trim(path, "/"), "/")
	reqID := parts[0]

	if len(parts) == 1 {
		// GET /access-requests/{id}
		if r.Method == http.MethodGet {
			accessRequests.mu.RLock()
			ar, ok := accessRequests.requests[reqID]
			accessRequests.mu.RUnlock()
			if !ok {
				writeJSONError(w, http.StatusNotFound, "access request not found")
				return
			}
			writeJSON(w, http.StatusOK, ar)
			return
		}
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "approve":
			s.reviewAccessRequest(w, r, reqID, "approved")
			return
		case "reject":
			s.reviewAccessRequest(w, r, reqID, "rejected")
			return
		}
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) handleAccessRequestsPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")

	accessRequests.mu.RLock()
	defer accessRequests.mu.RUnlock()

	result := []*AccessRequest{}
	for _, ar := range accessRequests.requests {
		if ar.Status != "pending" {
			continue
		}
		if tenantID != "" && ar.TenantID != tenantID {
			continue
		}
		result = append(result, ar)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"requests": result,
		"count":    len(result),
	})
}

func (s *HTTPServer) createAccessRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID      string `json:"tenant_id"`
		RequesterID   string `json:"requester_id"`
		RoleID        string `json:"role_id"`
		Justification string `json:"justification"`
		ExpiryHours   int    `json:"expiry_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.RequesterID == "" {
		writeJSONError(w, http.StatusBadRequest, "requester_id is required")
		return
	}
	if req.RoleID == "" {
		writeJSONError(w, http.StatusBadRequest, "role_id is required")
		return
	}

	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 72 // default 72-hour expiry
	}

	now := time.Now().UTC()
	ar := &AccessRequest{
		ID:            uuid.New().String(),
		TenantID:      req.TenantID,
		RequesterID:   req.RequesterID,
		RoleID:        req.RoleID,
		Justification: req.Justification,
		Status:        "pending",
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Duration(req.ExpiryHours) * time.Hour),
	}

	accessRequests.mu.Lock()
	accessRequests.requests[ar.ID] = ar
	accessRequests.mu.Unlock()

	writeJSON(w, http.StatusCreated, ar)
}

func (s *HTTPServer) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	requesterID := r.URL.Query().Get("requester_id")
	tenantID := r.URL.Query().Get("tenant_id")

	accessRequests.mu.RLock()
	defer accessRequests.mu.RUnlock()

	result := []*AccessRequest{}
	for _, ar := range accessRequests.requests {
		if status != "" && ar.Status != status {
			continue
		}
		if requesterID != "" && ar.RequesterID != requesterID {
			continue
		}
		if tenantID != "" && ar.TenantID != tenantID {
			continue
		}
		result = append(result, ar)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"requests": result,
		"count":    len(result),
	})
}

func (s *HTTPServer) reviewAccessRequest(w http.ResponseWriter, r *http.Request, reqID, decision string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ApproverID string `json:"approver_id"`
		Note       string `json:"review_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }
	if req.ApproverID == "" {
		req.ApproverID = r.URL.Query().Get("approver_id")
	}
	if req.ApproverID == "" {
		writeJSONError(w, http.StatusBadRequest, "approver_id is required")
		return
	}

	accessRequests.mu.Lock()
	defer accessRequests.mu.Unlock()

	ar, ok := accessRequests.requests[reqID]
	if !ok {
		writeJSONError(w, http.StatusNotFound, "access request not found")
		return
	}
	if ar.Status != "pending" {
		writeJSONError(w, http.StatusConflict, "access request already reviewed")
		return
	}

	now := time.Now().UTC()
	ar.Status = decision
	ar.ApproverID = req.ApproverID
	ar.ReviewedAt = &now
	ar.ReviewNote = req.Note

	writeJSON(w, http.StatusOK, ar)
}
