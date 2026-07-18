package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HREvent represents a single employee change from an HR system.
type HREvent struct {
	EventType  string         `json:"event_type"` // hired, terminated, dept_change, manager_change
	EmployeeID string         `json:"employee_id"`
	Email      string         `json:"email"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Department string         `json:"department"`
	Manager    string         `json:"manager"`
	Title      string         `json:"title"`
	Status     string         `json:"status"` // active, inactive
	Details    map[string]any `json:"details,omitempty"`
}

// HRConnectorConfig defines an HR system connection.
type HRConnectorConfig struct {
	ID         uuid.UUID      `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"` // workday, bamboohr, csv
	Config     map[string]any `json:"config"`
	Enabled    bool           `json:"enabled"`
	LastSyncAt *time.Time     `json:"last_sync_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// hrConnectorRepo manages HR connectors + sync logs in PostgreSQL.
type hrConnectorRepo struct {
	pool *pgxpool.Pool
}

func newHRConnectorRepo(pool *pgxpool.Pool) *hrConnectorRepo {
	return &hrConnectorRepo{pool: pool}
}

func (r *hrConnectorRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS hr_connectors (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID, name TEXT NOT NULL, type TEXT NOT NULL,
			config JSONB DEFAULT '{}', enabled BOOLEAN DEFAULT TRUE,
			last_sync_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS hr_sync_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			connector_id UUID NOT NULL REFERENCES hr_connectors(id) ON DELETE CASCADE,
			source TEXT NOT NULL, event_type TEXT NOT NULL,
			employee_id TEXT NOT NULL, ggid_user_id TEXT,
			status TEXT DEFAULT 'pending', details JSONB DEFAULT '{}',
			synced_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_hr_sync_connector ON hr_sync_log(connector_id, synced_at DESC);
	`)
	return err
}

func (r *hrConnectorRepo) CreateConnector(ctx context.Context, c *HRConnectorConfig) error {
	if r.pool == nil { return nil }
	if c.ID == uuid.Nil { c.ID = uuid.New() }
	cfgJSON, _ := json.Marshal(c.Config)
	_, err := r.pool.Exec(ctx, `INSERT INTO hr_connectors (id,name,type,config,enabled) VALUES ($1,$2,$3,$4,$5)`,
		c.ID, c.Name, c.Type, cfgJSON, c.Enabled)
	return err
}

func (r *hrConnectorRepo) ListConnectors(ctx context.Context) ([]*HRConnectorConfig, error) {
	if r.pool == nil { return []*HRConnectorConfig{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT id,name,type,config,enabled,last_sync_at,created_at FROM hr_connectors ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*HRConnectorConfig
	for rows.Next() {
		c := &HRConnectorConfig{}
		var cfgJSON []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &cfgJSON, &c.Enabled, &c.LastSyncAt, &c.CreatedAt); err != nil { continue }
		json.Unmarshal(cfgJSON, &c.Config)
		result = append(result, c)
	}
	return result, nil
}

func (r *hrConnectorRepo) LogSyncEvent(ctx context.Context, connectorID uuid.UUID, event *HREvent, ggidUserID string) {
	if r.pool == nil { return }
	detailsJSON, _ := json.Marshal(event.Details)
	r.pool.Exec(ctx, `INSERT INTO hr_sync_log (connector_id,source,event_type,employee_id,ggid_user_id,status,details) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		connectorID, "hr_sync", event.EventType, event.EmployeeID, ggidUserID, "processed", detailsJSON)
}

func (r *hrConnectorRepo) ListSyncLog(ctx context.Context, limit int) ([]map[string]any, error) {
	if r.pool == nil { return []map[string]any{}, nil }
	if limit <= 0 || limit > 100 { limit = 50 }
	rows, err := r.pool.Query(ctx, `SELECT id,connector_id,source,event_type,employee_id,ggid_user_id,status,synced_at FROM hr_sync_log ORDER BY synced_at DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		m := map[string]any{}
		var id, connID, source, eventType, employeeID, ggidUserID, status string
		var syncedAt time.Time
		if err := rows.Scan(&id, &connID, &source, &eventType, &employeeID, &ggidUserID, &status, &syncedAt); err != nil { continue }
		m["id"] = id; m["connector_id"] = connID; m["source"] = source
		m["event_type"] = eventType; m["employee_id"] = employeeID
		m["ggid_user_id"] = ggidUserID; m["status"] = status; m["synced_at"] = syncedAt
		result = append(result, m)
	}
	return result, nil
}

// --- Simulated Connector Implementations ---

// SyncHREvents simulates pulling events from the configured HR system.
// In production, this would call Workday/BambooHR APIs.
func syncHREvents(connector *HRConnectorConfig) []HREvent {
	// Placeholder: in production, each connector type would make real API calls.
	// workday: GET /api/v1/workday/employees changed since last_sync
	// bamboohr: GET /api/gateway.php/{domain}/v1/reports/custom
	// csv: parse uploaded file
	return []HREvent{} // no events in simulation mode
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleHRConnectors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req HRConnectorConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.Type == "" {
			writeJSONError(w, http.StatusBadRequest, "name and type required")
			return
		}
		validTypes := map[string]bool{"workday": true, "bamboohr": true, "csv": true}
		if !validTypes[req.Type] {
			writeJSONError(w, http.StatusBadRequest, "type must be workday, bamboohr, or csv")
			return
		}
		req.Enabled = true
		if h.hrConnectorRepo != nil {
			h.hrConnectorRepo.CreateConnector(r.Context(), &req)
		}
		writeJSON(w, http.StatusCreated, req)
	case http.MethodGet:
		var connectors []*HRConnectorConfig
		if h.hrConnectorRepo != nil {
			connectors, _ = h.hrConnectorRepo.ListConnectors(r.Context())
		}
		if connectors == nil { connectors = []*HRConnectorConfig{} }
		writeJSON(w, http.StatusOK, map[string]any{"connectors": connectors, "count": len(connectors)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) handleHRSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Get all enabled connectors and sync each.
	connectors, _ := h.hrConnectorRepo.ListConnectors(r.Context())
	totalEvents := 0
	for _, c := range connectors {
		if !c.Enabled { continue }
		events := syncHREvents(c)
		for _, event := range events {
			h.hrConnectorRepo.LogSyncEvent(r.Context(), c.ID, &event, "")
			totalEvents++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "synced", "connectors_synced": len(connectors),
		"events_processed": totalEvents, "synced_at": time.Now().UTC(),
	})
}

func (h *HTTPHandler) handleHRSyncLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	var log []map[string]any
	if h.hrConnectorRepo != nil {
		log, _ = h.hrConnectorRepo.ListSyncLog(r.Context(), limit)
	}
	if log == nil { log = []map[string]any{} }
	writeJSON(w, http.StatusOK, map[string]any{"log": log, "count": len(log)})
}

func (h *HTTPHandler) handleHRDormant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var dormant []*UserLifecycleState
	if h.dormantRepo != nil {
		dormant, _ = h.dormantRepo.ListDormant(r.Context())
	}
	if dormant == nil { dormant = []*UserLifecycleState{} }
	writeJSON(w, http.StatusOK, map[string]any{"dormant_accounts": dormant, "count": len(dormant)})
}

func (h *HTTPHandler) handleHRReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var ghosts []*GhostAccount
	if h.dormantRepo != nil {
		ghosts, _ = h.dormantRepo.ListGhosts(r.Context())
	}
	if ghosts == nil { ghosts = []*GhostAccount{} }
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "reconciled", "ghosts_found": len(ghosts),
		"ghost_accounts": ghosts, "reconciled_at": time.Now().UTC(),
	})
}

func (h *HTTPHandler) SetHRConnectorRepo(repo *hrConnectorRepo) {
	h.hrConnectorRepo = repo
}

func (h *HTTPHandler) SetDormantRepo(repo *dormantRepo) {
	h.dormantRepo = repo
}
