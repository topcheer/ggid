package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

// JourneyDefinition represents an identity orchestration journey.
type JourneyDefinition struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Definition  string    `json:"definition"` // YAML JDL
	Status      string    `json:"status"`     // draft, active, archived
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JDL represents parsed Journey Definition Language YAML.
type JDL struct {
	Steps []JDLStep `yaml:"steps"`
}

type JDLStep struct {
	ID        string         `yaml:"id"`
	Name      string         `yaml:"name"`
	Action    string         `yaml:"action"`   // assign_role, revoke_access, notify, etc.
	Condition string         `yaml:"condition"` // CEL-like expression (e.g., "user.department == 'eng'")
	Params    map[string]any `yaml:"params"`
}

// JourneyDryRunResult is the output of a dry-run simulation.
type JourneyDryRunResult struct {
	JourneyID string       `json:"journey_id"`
	Success   bool         `json:"success"`
	Steps     []DryRunStep `json:"steps"`
	Errors    []string     `json:"errors,omitempty"`
}

type DryRunStep struct {
	Step    string `json:"step"`
	Name    string `json:"name"`
	Action  string `json:"action"`
	Result  string `json:"result"` // success, skipped, error
	Message string `json:"message,omitempty"`
}

// journeyRepo manages journey persistence in PostgreSQL.
type journeyRepo struct {
	pool *pgxpool.Pool
}

func newJourneyRepo(pool *pgxpool.Pool) *journeyRepo {
	return &journeyRepo{pool: pool}
}

func (r *journeyRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS journeys (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID NOT NULL,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			definition  TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'draft',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_journeys_tenant ON journeys (tenant_id, status);
	`)
	return err
}

func (r *journeyRepo) Create(ctx context.Context, j *JourneyDefinition) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO journeys (id, tenant_id, name, description, definition, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
		j.ID, j.TenantID, j.Name, j.Description, j.Definition, j.Status, j.CreatedAt)
	return err
}

func (r *journeyRepo) Get(ctx context.Context, id string) (*JourneyDefinition, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id::text, name, description, definition, status, created_at, updated_at FROM journeys WHERE id = $1`, id)
	var j JourneyDefinition
	if err := row.Scan(&j.ID, &j.TenantID, &j.Name, &j.Description, &j.Definition, &j.Status, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &j, nil
}

func (r *journeyRepo) List(ctx context.Context, tenantID, status string) ([]*JourneyDefinition, error) {
	if r.pool == nil {
		return []*JourneyDefinition{}, nil
	}
	q := `SELECT id, tenant_id::text, name, description, definition, status, created_at, updated_at FROM journeys WHERE tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		q += ` AND status = $2`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*JourneyDefinition
	for rows.Next() {
		var j JourneyDefinition
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Name, &j.Description, &j.Definition, &j.Status, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		result = append(result, &j)
	}
	return result, nil
}

func (r *journeyRepo) Update(ctx context.Context, j *JourneyDefinition) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE journeys SET name=$2, description=$3, definition=$4, status=$5, updated_at=now() WHERE id=$1`,
		j.ID, j.Name, j.Description, j.Definition, j.Status)
	return err
}

func (r *journeyRepo) Delete(ctx context.Context, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM journeys WHERE id=$1`, id)
	return err
}

// handleJourneys routes Journey Definition CRUD + dry-run.
func (h *HTTPHandler) handleJourneys(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/dry-run") {
		parts := strings.Split(strings.TrimSuffix(path, "/dry-run"), "/")
		id := parts[len(parts)-1]
		h.journeyDryRun(w, r, id)
		return
	}

	if path == "/api/v1/identity/journeys" || strings.HasSuffix(path, "/journeys") {
		switch r.Method {
		case http.MethodGet:
			h.journeyList(w, r)
		case http.MethodPost:
			h.journeyCreate(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) >= 5 {
		id := parts[len(parts)-1]
		switch r.Method {
		case http.MethodGet:
			h.journeyGet(w, r, id)
		case http.MethodPut:
			h.journeyUpdate(w, r, id)
		case http.MethodDelete:
			h.journeyDelete(w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (h *HTTPHandler) journeyCreate(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	var j JourneyDefinition
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if j.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	j.ID = uuid.New().String()
	if tc != nil {
		j.TenantID = tc.TenantID.String()
	}
	if j.Status == "" {
		j.Status = "draft"
	}
	j.CreatedAt = time.Now()
	j.UpdatedAt = j.CreatedAt
	if h.journeyRepo != nil {
		if err := h.journeyRepo.Create(r.Context(), &j); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create journey")
			return
		}
	}
	writeJSON(w, http.StatusCreated, j)
}

func (h *HTTPHandler) journeyList(w http.ResponseWriter, r *http.Request) {
	tc, _ := ggidtenant.FromContext(r.Context())
	status := r.URL.Query().Get("status")
	var journeys []*JourneyDefinition
	if h.journeyRepo != nil && tc != nil {
		var err error
		journeys, err = h.journeyRepo.List(r.Context(), tc.TenantID.String(), status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list")
			return
		}
	}
	if journeys == nil {
		journeys = []*JourneyDefinition{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"journeys": journeys, "total": len(journeys)})
}

func (h *HTTPHandler) journeyGet(w http.ResponseWriter, r *http.Request, id string) {
	var j *JourneyDefinition
	if h.journeyRepo != nil {
		var err error
		j, err = h.journeyRepo.Get(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "journey not found")
			return
		}
	}
	if j == nil {
		writeError(w, http.StatusNotFound, "journey not found")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *HTTPHandler) journeyUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var update JourneyDefinition
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	update.ID = id
	if h.journeyRepo != nil {
		if err := h.journeyRepo.Update(r.Context(), &update); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update")
			return
		}
	}
	writeJSON(w, http.StatusOK, update)
}

func (h *HTTPHandler) journeyDelete(w http.ResponseWriter, r *http.Request, id string) {
	if h.journeyRepo != nil {
		h.journeyRepo.Delete(r.Context(), id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// journeyDryRun parses JDL YAML and simulates step execution.
func (h *HTTPHandler) journeyDryRun(w http.ResponseWriter, r *http.Request, id string) {
	var j *JourneyDefinition
	if h.journeyRepo != nil {
		var err error
		j, err = h.journeyRepo.Get(r.Context(), id)
		if err != nil || j == nil {
			writeError(w, http.StatusNotFound, "journey not found")
			return
		}
	}
	if j == nil {
		writeError(w, http.StatusNotFound, "journey not found")
		return
	}

	result := JourneyDryRunResult{JourneyID: id, Steps: []DryRunStep{}}

	// Parse YAML definition.
	var jdl JDL
	if j.Definition == "" {
		result.Success = false
		result.Errors = []string{"no JDL definition provided"}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if err := yaml.Unmarshal([]byte(j.Definition), &jdl); err != nil {
		result.Success = false
		result.Errors = []string{fmt.Sprintf("YAML parse error: %v", err)}
		writeJSON(w, http.StatusOK, result)
		return
	}

	// Simulate each step.
	result.Success = true
	for _, step := range jdl.Steps {
		ds := DryRunStep{
			Step:   step.ID,
			Name:   step.Name,
			Action: step.Action,
			Result: "success",
		}

		// Evaluate condition (simplified CEL-like: key == value).
		if step.Condition != "" {
			if !evaluateJDLCondition(step.Condition, r.URL.Query()) {
				ds.Result = "skipped"
				ds.Message = fmt.Sprintf("condition not met: %s", step.Condition)
			}
		}

		// Validate action type.
		validActions := map[string]bool{
			"assign_role": true, "revoke_access": true, "notify": true,
			"create_account": true, "disable_account": true,
		}
		if !validActions[step.Action] {
			ds.Result = "error"
			ds.Message = fmt.Sprintf("unknown action: %s", step.Action)
			result.Success = false
		} else if ds.Message == "" {
			ds.Message = fmt.Sprintf("would execute %s", step.Action)
		}

		result.Steps = append(result.Steps, ds)
	}

	writeJSON(w, http.StatusOK, result)
}

// evaluateJDLCondition evaluates a simple CEL-like expression.
// Supports: key == 'value' or key == value
func evaluateJDLCondition(expr string, params interface{ Get(string) string }) bool {
	// Parse "key == 'value'" pattern.
	parts := strings.SplitN(expr, "==", 2)
	if len(parts) != 2 {
		return true // can't parse → assume true
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	val = strings.Trim(val, "'\"")
	// Check against query params (dry-run context).
	actual := params.Get(key)
	return actual == val
}

// SetJourneyRepo injects the journey repository.
func (h *HTTPHandler) SetJourneyRepo(repo *journeyRepo) {
	h.journeyRepo = repo
}

// suppress unused
var _ = sql.ErrNoRows
var _ = log.Printf
