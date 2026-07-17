package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AccessRequest represents a user's request for elevated access.
type AccessRequest struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	RequesterID   string     `json:"requester_id"`
	RoleID        string     `json:"role_id"`
	Justification string     `json:"justification"`
	Status        string     `json:"status"`
	ApproverID    string     `json:"approver_id,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote    string     `json:"review_note,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
}

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

	parts := strings.Split(strings.Trim(path, "/"), "/")
	reqID := parts[0]

	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			if s.policyMap != nil {
				ar, _ := s.policyMap.Get(r.Context(), "access_requests_store", reqID)
				if ar != nil {
					writeJSON(w, http.StatusOK, ar)
					return
				}
			}
			writeJSONError(w, http.StatusNotFound, "access request not found")
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
	var result []map[string]any
	if s.policyMap != nil {
		rows, _ := s.policyMap.List(r.Context(), "access_requests_store")
		for _, row := range rows {
			if pmGetString(row, "status") != "pending" {
				continue
			}
			if tenantID != "" && pmGetString(row, "tenant_id") != tenantID {
				continue
			}
			result = append(result, row)
		}
	}
	if result == nil {
		result = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": result, "count": len(result)})
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
		req.ExpiryHours = 72
	}
	now := time.Now().UTC()
	ar := &AccessRequest{
		ID: uuid.New().String(), TenantID: req.TenantID, RequesterID: req.RequesterID,
		RoleID: req.RoleID, Justification: req.Justification, Status: "pending",
		CreatedAt: now, ExpiresAt: now.Add(time.Duration(req.ExpiryHours) * time.Hour),
	}
	if s.policyMap != nil {
		s.policyMap.Store(r.Context(), "access_requests_store", ar.ID, map[string]any{
			"tenant_id": ar.TenantID, "requester_id": ar.RequesterID, "role_id": ar.RoleID,
			"justification": ar.Justification, "status": ar.Status,
			"created_at": ar.CreatedAt, "expires_at": ar.ExpiresAt,
		})
	}
	writeJSON(w, http.StatusCreated, ar)
}

func (s *HTTPServer) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	requesterID := r.URL.Query().Get("requester_id")
	tenantID := r.URL.Query().Get("tenant_id")
	var result []map[string]any
	if s.policyMap != nil {
		rows, _ := s.policyMap.List(r.Context(), "access_requests_store")
		for _, row := range rows {
			if status != "" && pmGetString(row, "status") != status {
				continue
			}
			if requesterID != "" && pmGetString(row, "requester_id") != requesterID {
				continue
			}
			if tenantID != "" && pmGetString(row, "tenant_id") != tenantID {
				continue
			}
			result = append(result, row)
		}
	}
	if result == nil {
		result = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": result, "count": len(result)})
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
	json.NewDecoder(r.Body).Decode(&req)
	if s.policyMap != nil {
		existing, _ := s.policyMap.Get(r.Context(), "access_requests_store", reqID)
		if existing == nil {
			writeJSONError(w, http.StatusNotFound, "access request not found")
			return
		}
		existing["status"] = decision
		existing["approver_id"] = req.ApproverID
		existing["review_note"] = req.Note
		existing["reviewed_at"] = time.Now().UTC()
		s.policyMap.Store(r.Context(), "access_requests_store", reqID, existing)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": decision, "id": reqID})
}
