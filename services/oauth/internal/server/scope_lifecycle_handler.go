package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ScopeLifecycle struct {
	ScopeID        string   `json:"scope_id"`
	RequestedScope string   `json:"requested_scope"`
	Requester      string   `json:"requester"`
	ApproverChain  []string `json:"approver_chain"`
	Status         string   `json:"status"`
	RiskLevel      string   `json:"risk_level"`
	AutoExpireDays int      `json:"auto_expire_days"`
	CreatedAt      string   `json:"created_at"`
}

type ScopeLifecycleRequest struct {
	RequestedScope string   `json:"requested_scope"`
	Requester      string   `json:"requester"`
	ApproverChain  []string `json:"approver_chain"`
	RiskLevel      string   `json:"risk_level"`
	AutoExpireDays int      `json:"auto_expire_days"`
}

var scopeLifecycleStore sync.Map

func handleScopeLifecycle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req ScopeLifecycleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.RequestedScope == "" {
			req.RequestedScope = "unknown:scope"
		}
		if req.Requester == "" {
			req.Requester = "system"
		}
		if len(req.ApproverChain) == 0 {
			req.ApproverChain = []string{"admin@ggid.dev"}
		}
		if req.RiskLevel == "" {
			req.RiskLevel = "medium"
		}
		if req.AutoExpireDays == 0 {
			req.AutoExpireDays = 90
		}
		sl := ScopeLifecycle{
			ScopeID:        fmt.Sprintf("scope-lc-%d", time.Now().UnixNano()%100000),
			RequestedScope: req.RequestedScope,
			Requester:      req.Requester,
			ApproverChain:  req.ApproverChain,
			Status:         "pending",
			RiskLevel:      req.RiskLevel,
			AutoExpireDays: req.AutoExpireDays,
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		}
		scopeLifecycleStore.Store(sl.ScopeID, sl)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sl)
	case http.MethodGet:
		var items []ScopeLifecycle
		scopeLifecycleStore.Range(func(_, v any) bool {
			items = append(items, v.(ScopeLifecycle))
			return true
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"scopes": items, "count": len(items)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
