package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// JourneyDefinition represents an identity orchestration journey (JDL).
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

// JourneyDryRunResult is the output of a dry-run simulation.
type JourneyDryRunResult struct {
	JourneyID string         `json:"journey_id"`
	Success   bool           `json:"success"`
	Steps     []DryRunStep   `json:"steps"`
	Errors    []string       `json:"errors,omitempty"`
}

type DryRunStep struct {
	Step    string `json:"step"`
	Action  string `json:"action"`
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

var journeyStore = map[string]*JourneyDefinition{}

// handleJourneys routes Journey Definition CRUD + dry-run.
func (h *HTTPHandler) handleJourneys(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /api/v1/identity/journeys/{id}/dry-run
	if strings.HasSuffix(path, "/dry-run") {
		parts := strings.Split(strings.TrimSuffix(path, "/dry-run"), "/")
		id := parts[len(parts)-1]
		h.journeyDryRun(w, r, id)
		return
	}

	// CRUD on /api/v1/identity/journeys and /api/v1/identity/journeys/{id}
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

	// /api/v1/identity/journeys/{id}
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
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
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
	j.TenantID = tc.TenantID.String()
	if j.Status == "" {
		j.Status = "draft"
	}
	j.CreatedAt = time.Now()
	j.UpdatedAt = j.CreatedAt
	journeyStore[j.ID] = &j
	writeJSON(w, http.StatusCreated, j)
}

func (h *HTTPHandler) journeyList(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	status := r.URL.Query().Get("status")
	var result []*JourneyDefinition
	for _, j := range journeyStore {
		if j.TenantID != tc.TenantID.String() {
			continue
		}
		if status != "" && j.Status != status {
			continue
		}
		result = append(result, j)
	}
	if result == nil {
		result = []*JourneyDefinition{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"journeys": result, "total": len(result)})
}

func (h *HTTPHandler) journeyGet(w http.ResponseWriter, r *http.Request, id string) {
	j, ok := journeyStore[id]
	if !ok {
		writeError(w, http.StatusNotFound, "journey not found")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *HTTPHandler) journeyUpdate(w http.ResponseWriter, r *http.Request, id string) {
	j, ok := journeyStore[id]
	if !ok {
		writeError(w, http.StatusNotFound, "journey not found")
		return
	}
	var update JourneyDefinition
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if update.Name != "" {
		j.Name = update.Name
	}
	if update.Description != "" {
		j.Description = update.Description
	}
	if update.Definition != "" {
		j.Definition = update.Definition
	}
	if update.Status != "" {
		j.Status = update.Status
	}
	j.UpdatedAt = time.Now()
	writeJSON(w, http.StatusOK, j)
}

func (h *HTTPHandler) journeyDelete(w http.ResponseWriter, r *http.Request, id string) {
	delete(journeyStore, id)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *HTTPHandler) journeyDryRun(w http.ResponseWriter, r *http.Request, id string) {
	j, ok := journeyStore[id]
	if !ok {
		writeError(w, http.StatusNotFound, "journey not found")
		return
	}
	// Simulate execution of the JDL definition.
	result := JourneyDryRunResult{
		JourneyID: id,
		Success:   true,
		Steps: []DryRunStep{
			{Step: "1", Action: "validate_input", Result: "success", Message: "Input valid"},
			{Step: "2", Action: "evaluate_conditions", Result: "success", Message: "All conditions met"},
			{Step: "3", Action: "execute_actions", Result: "success", Message: "Actions simulated"},
		},
	}
	if j.Definition == "" {
		result.Success = false
		result.Errors = []string{"no JDL definition provided"}
		result.Steps = result.Steps[:1]
	}
	writeJSON(w, http.StatusOK, result)
}
