package server

import (
	"encoding/json"
	"net/http"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// ListObjectsRequest finds all objects a subject can access with a given relation.
type ListObjectsRequest struct {
	Namespace string `json:"namespace"`
	Relation  string `json:"relation"`
	Subject   string `json:"subject"`
}

// ListSubjectsRequest finds all subjects with a relation on an object.
type ListSubjectsRequest struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
}

// handleReBACListObjects returns all objects where subject has the relation.
// POST /api/v1/identity/list-objects
func (h *HTTPHandler) handleReBACListObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req ListObjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Namespace == "" || req.Relation == "" || req.Subject == "" {
		writeJSONError(w, http.StatusBadRequest, "namespace, relation, subject required")
		return
	}

	if h.rebacRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{"relations": []any{}, "count": 0})
		return
	}

	tuples, err := h.rebacRepo.ReadTuples(r.Context(), tc.TenantID, req.Namespace, "", req.Relation, req.Subject)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to query tuples")
		return
	}

	objects := make([]string, 0, len(tuples))
	for _, t := range tuples {
		objects = append(objects, t.Namespace+":"+t.Object)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"objects": objects,
		"total":   len(objects),
	})
}

// handleReBACListSubjects returns all subjects with the relation on an object.
// POST /api/v1/identity/list-subjects
func (h *HTTPHandler) handleReBACListSubjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req ListSubjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Namespace == "" || req.Object == "" || req.Relation == "" {
		writeJSONError(w, http.StatusBadRequest, "namespace, object, relation required")
		return
	}

	if h.rebacRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{"relations": []any{}, "count": 0})
		return
	}

	subjects, err := h.rebacRepo.DirectSubjects(r.Context(), tc.TenantID, req.Namespace, req.Object, req.Relation)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to query subjects")
		return
	}

	// Also check computed relations (can_view → viewer, editor, owner).
	computedRels := computedRelationsFor(req.Relation)
	for _, altRel := range computedRels {
		altSubjects, _ := h.rebacRepo.DirectSubjects(r.Context(), tc.TenantID, req.Namespace, req.Object, altRel)
		subjects = appendUnique(subjects, altSubjects)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subjects": subjects,
		"total":    len(subjects),
	})
}

func appendUnique(slice, items []string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}
