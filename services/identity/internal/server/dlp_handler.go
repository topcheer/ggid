package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DLPPolicy defines a Data Loss Prevention rule.
type DLPPolicy struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Trigger     string         `json:"trigger"` // export, download, share, api_call
	Conditions map[string]any `json:"conditions"`
	Action      string         `json:"action"` // block, mask, log
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// DLPEvent records a DLP enforcement action.
type DLPEvent struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	PolicyID          *uuid.UUID `json:"policy_id,omitempty"`
	UserID            string    `json:"user_id,omitempty"`
	UserName          string    `json:"user_name,omitempty"`
	Trigger           string    `json:"trigger"`
	ResourceType      string    `json:"resource_type,omitempty"`
	DataClassification string  `json:"data_classification,omitempty"`
	ActionTaken       string    `json:"action_taken"`
	Reason            string    `json:"reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// DLPTestResult is the outcome of a policy simulation.
type DLPTestResult struct {
	Matched            bool   `json:"matched"`
	PolicyID           string `json:"policy_id,omitempty"`
	Action             string `json:"action,omitempty"`
	Reason             string `json:"reason,omitempty"`
	DataClassification string `json:"data_classification,omitempty"`
}

// dlpRepo manages DLP policies + events in PostgreSQL.
type dlpRepo struct {
	pool *pgxpool.Pool
}

func newDLPRepo(pool *pgxpool.Pool) *dlpRepo {
	return &dlpRepo{pool: pool}
}

func (r *dlpRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS dlp_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, name TEXT NOT NULL, description TEXT DEFAULT '',
			trigger TEXT NOT NULL, conditions JSONB DEFAULT '{}',
			action TEXT NOT NULL DEFAULT 'log', enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_dlp_policies_tenant ON dlp_policies(tenant_id, enabled);
		CREATE TABLE IF NOT EXISTS dlp_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, policy_id UUID,
			user_id TEXT, user_name TEXT, trigger TEXT NOT NULL,
			resource_type TEXT, data_classification TEXT,
			action_taken TEXT NOT NULL, reason TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_dlp_events_tenant ON dlp_events(tenant_id, created_at DESC);
	`)
	return err
}

func (r *dlpRepo) Create(ctx context.Context, p *DLPPolicy) error {
	if r.pool == nil { return nil }
	if p.ID == uuid.Nil { p.ID = uuid.New() }
	condJSON, _ := json.Marshal(p.Conditions)
	_, err := r.pool.Exec(ctx, `INSERT INTO dlp_policies (id,tenant_id,name,description,trigger,conditions,action,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.TenantID, p.Name, p.Description, p.Trigger, condJSON, p.Action, p.Enabled)
	return err
}

func (r *dlpRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*DLPPolicy, error) {
	if r.pool == nil { return []*DLPPolicy{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT id,name,description,trigger,conditions,action,enabled,created_at,updated_at FROM dlp_policies WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*DLPPolicy
	for rows.Next() {
		var p DLPPolicy
		var condJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Trigger, &condJSON, &p.Action, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil { continue }
		json.Unmarshal(condJSON, &p.Conditions)
		result = append(result, &p)
	}
	return result, nil
}

func (r *dlpRepo) Update(ctx context.Context, p *DLPPolicy) error {
	if r.pool == nil { return nil }
	condJSON, _ := json.Marshal(p.Conditions)
	_, err := r.pool.Exec(ctx, `UPDATE dlp_policies SET name=$2,description=$3,trigger=$4,conditions=$5,action=$6,enabled=$7,updated_at=now() WHERE id=$1`,
		p.ID, p.Name, p.Description, p.Trigger, condJSON, p.Action, p.Enabled)
	return err
}

func (r *dlpRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `DELETE FROM dlp_policies WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

func (r *dlpRepo) LogEvent(ctx context.Context, e *DLPEvent) {
	if r.pool == nil { return }
	if e.ID == uuid.Nil { e.ID = uuid.New() }
	r.pool.Exec(ctx, `INSERT INTO dlp_events (id,tenant_id,policy_id,user_id,user_name,trigger,resource_type,data_classification,action_taken,reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		e.ID, e.TenantID, e.PolicyID, e.UserID, e.UserName, e.Trigger, e.ResourceType, e.DataClassification, e.ActionTaken, e.Reason)
}

func (r *dlpRepo) ListEvents(ctx context.Context, tenantID uuid.UUID, action string) ([]*DLPEvent, error) {
	if r.pool == nil { return []*DLPEvent{}, nil }
	q := `SELECT id,tenant_id,policy_id,user_id,user_name,trigger,resource_type,data_classification,action_taken,reason,created_at FROM dlp_events WHERE tenant_id=$1`
	args := []any{tenantID}
	if action != "" { q += ` AND action_taken=$2`; args = append(args, action) }
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*DLPEvent
	for rows.Next() {
		var e DLPEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.PolicyID, &e.UserID, &e.UserName, &e.Trigger, &e.ResourceType, &e.DataClassification, &e.ActionTaken, &e.Reason, &e.CreatedAt); err != nil { continue }
		result = append(result, &e)
	}
	return result, nil
}

// --- DLP Engine ---

// EvaluateDLP checks policies against a request context.
// Returns matched policy + action (block/mask/log) or nil if no match.
func EvaluateDLP(policies []*DLPPolicy, trigger, resourceType, dataClassification, userRole string) *DLPTestResult {
	for _, p := range policies {
		if !p.Enabled || p.Trigger != trigger {
			continue
		}
		// Evaluate conditions.
		if matchesDLPConditions(p.Conditions, dataClassification, userRole) {
			return &DLPTestResult{
				Matched:            true,
				PolicyID:           p.ID.String(),
				Action:             p.Action,
				Reason:             p.Name,
				DataClassification: dataClassification,
			}
		}
	}
	// Default action based on classification (admins bypass defaults).
	switch dataClassification {
	case "core":
		if userRole == "admin" {
			return &DLPTestResult{Matched: false, Action: "log", DataClassification: "core"}
		}
		return &DLPTestResult{Matched: true, Action: "block", Reason: "core data requires admin", DataClassification: "core"}
	case "important":
		if userRole == "admin" {
			return &DLPTestResult{Matched: false, Action: "log", DataClassification: "important"}
		}
		return &DLPTestResult{Matched: true, Action: "mask", Reason: "important data PII masking", DataClassification: "important"}
	default:
		return &DLPTestResult{Matched: false, Action: "log"}
	}
}

func matchesDLPConditions(conditions map[string]any, classification, userRole string) bool {
	if len(conditions) == 0 { return true }
	andConds, ok := conditions["and"].([]any)
	if !ok { return true }
	for _, cond := range andConds {
		condMap, ok := cond.(map[string]any)
		if !ok { continue }
		for key, expected := range condMap {
			switch key {
			case "$data.classification":
				if fmt.Sprintf("%v", expected) != classification { return false }
			case "$user.role":
				if expMap, ok := expected.(map[string]any); ok {
					if ne, has := expMap["$ne"]; has && fmt.Sprintf("%v", ne) == userRole { return false }
				} else if fmt.Sprintf("%v", expected) != userRole { return false }
			}
		}
	}
	return true
}

// --- API Handlers ---

func (h *HTTPHandler) handleDLP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	tc, _ := ggidtenant.FromContext(r.Context())

	if strings.HasSuffix(path, "/test") {
		h.dlpTestPolicy(w, r)
		return
	}
	if strings.HasSuffix(path, "/events") {
		h.dlpListEvents(w, r)
		return
	}
	if strings.HasSuffix(path, "/heatmap") {
		h.dlpHeatmap(w, r)
		return
	}

	// CRUD on policies
	switch r.Method {
	case http.MethodPost:
		var p DLPPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if tc != nil { p.TenantID = tc.TenantID }
		if p.Name == "" || p.Trigger == "" {
			writeJSONError(w, http.StatusBadRequest, "name and trigger required")
			return
		}
		if p.Action == "" { p.Action = "log" }
		p.Enabled = true
		if h.dlpPolicyRepo != nil {
			if err := h.dlpPolicyRepo.Create(r.Context(), &p); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed")
				return
			}
		}
		writeJSON(w, http.StatusCreated, p)
	case http.MethodGet:
		var policies []*DLPPolicy
		if h.dlpPolicyRepo != nil && tc != nil {
			policies, _ = h.dlpPolicyRepo.List(r.Context(), tc.TenantID)
		}
		if policies == nil { policies = []*DLPPolicy{} }
		writeJSON(w, http.StatusOK, map[string]any{"policies": policies, "total": len(policies)})
	case http.MethodPut:
		parts := strings.Split(path, "/")
		id, err := uuid.Parse(parts[len(parts)-1])
		if err != nil { writeJSONError(w, http.StatusBadRequest, "invalid id"); return }
		var p DLPPolicy
		json.NewDecoder(r.Body).Decode(&p)
		p.ID = id
		if tc != nil { p.TenantID = tc.TenantID }
		if h.dlpPolicyRepo != nil { h.dlpPolicyRepo.Update(r.Context(), &p) }
		writeJSON(w, http.StatusOK, p)
	case http.MethodDelete:
		parts := strings.Split(path, "/")
		id, err := uuid.Parse(parts[len(parts)-1])
		if err != nil { writeJSONError(w, http.StatusBadRequest, "invalid id"); return }
		if h.dlpPolicyRepo != nil && tc != nil { h.dlpPolicyRepo.Delete(r.Context(), id, tc.TenantID) }
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	}
}

func (h *HTTPHandler) dlpListEvents(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	action := r.URL.Query().Get("severity")
	if action == "" { action = r.URL.Query().Get("action") }
	var events []*DLPEvent
	if h.dlpPolicyRepo != nil && tc != nil {
		events, _ = h.dlpPolicyRepo.ListEvents(r.Context(), tc.TenantID, action)
	}
	if events == nil { events = []*DLPEvent{} }
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "total": len(events)})
}

func (h *HTTPHandler) dlpHeatmap(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	_ = tc
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": []map[string]any{},
		"total": 0,
		"generated_at": time.Now().UTC(),
	})
}

func (h *HTTPHandler) dlpTestPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserID       string `json:"user_id"`
		ResourceType string `json:"resource_type"`
		Trigger      string `json:"trigger"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	tc, _ := ggidtenant.FromContext(r.Context())
	userRole := r.Header.Get("X-User-Role")

	// Get classification from data governance repo.
	classification := "general"
	if h.dataGovRepo != nil && tc != nil {
		if dc, _ := h.dataGovRepo.LookupClassification(r.Context(), tc.TenantID, "user_attribute", req.ResourceType); dc != nil {
			classification = dc.Classification
		}
	}

	var policies []*DLPPolicy
	if h.dlpPolicyRepo != nil && tc != nil {
		policies, _ = h.dlpPolicyRepo.List(r.Context(), tc.TenantID)
	}

	result := EvaluateDLP(policies, req.Trigger, req.ResourceType, classification, userRole)

	// Log the test event.
	if h.dlpPolicyRepo != nil && tc != nil && result.Matched {
		var polID *uuid.UUID
		if pid, err := uuid.Parse(result.PolicyID); err == nil { polID = &pid }
		h.dlpPolicyRepo.LogEvent(r.Context(), &DLPEvent{
			TenantID: tc.TenantID, PolicyID: polID, UserID: req.UserID,
			Trigger: req.Trigger, ResourceType: req.ResourceType,
			DataClassification: classification, ActionTaken: result.Action, Reason: result.Reason,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) SetDLPRepo(repo *dlpRepo) {
	h.dlpPolicyRepo = repo
}
