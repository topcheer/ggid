package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type AuditTrailEntry struct {
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
	Detail    string `json:"detail"`
}

type PolicyException struct {
	ExceptionID       string            `json:"exception_id"`
	PolicyID          string            `json:"policy_id"`
	ExceptionReason   string            `json:"exception_reason"`
	GrantedTo         string            `json:"granted_to"`
	ExpiresAt         string            `json:"expires_at"`
	Approver          string            `json:"approver"`
	RiskOverrideLevel string            `json:"risk_override_level"`
	AuditTrail        []AuditTrailEntry `json:"audit_trail"`
	CreatedAt         string            `json:"created_at"`
}

type ExceptionRequest struct {
	PolicyID          string `json:"policy_id"`
	ExceptionReason   string `json:"exception_reason"`
	GrantedTo         string `json:"granted_to"`
	ExpiresAt         string `json:"expires_at"`
	Approver          string `json:"approver"`
	RiskOverrideLevel string `json:"risk_override_level"`
}

var exceptionStore sync.Map

func (s *HTTPServer) handlePolicyExceptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req ExceptionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.PolicyID == "" {
			req.PolicyID = "unknown"
		}
		if req.ExceptionReason == "" {
			req.ExceptionReason = "temporary access"
		}
		if req.RiskOverrideLevel == "" {
			req.RiskOverrideLevel = "medium"
		}
		if req.ExpiresAt == "" {
			req.ExpiresAt = time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
		}
		now := time.Now().UTC()
		exc := PolicyException{
			ExceptionID:       fmt.Sprintf("exc-%d", now.UnixNano()%100000),
			PolicyID:          req.PolicyID,
			ExceptionReason:   req.ExceptionReason,
			GrantedTo:         req.GrantedTo,
			ExpiresAt:         req.ExpiresAt,
			Approver:          req.Approver,
			RiskOverrideLevel: req.RiskOverrideLevel,
			AuditTrail: []AuditTrailEntry{
				{Action: "created", Actor: req.Approver, Timestamp: now.Format(time.RFC3339), Detail: "Exception granted"},
			},
			CreatedAt: now.Format(time.RFC3339),
		}
		exceptionStore.Store(exc.ExceptionID, exc)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(exc)
	case http.MethodGet:
		var items []PolicyException
		exceptionStore.Range(func(_, v any) bool {
			items = append(items, v.(PolicyException))
			return true
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"exceptions": items, "count": len(items)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
