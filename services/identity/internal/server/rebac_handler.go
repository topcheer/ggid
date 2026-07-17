package server

import (
	"encoding/json"
	"net/http"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleReBACTuples routes tuple CRUD.
// POST   /api/v1/identity/tuples — write tuple
// GET    /api/v1/identity/tuples — list tuples (filter by namespace/object/relation/subject)
// DELETE /api/v1/identity/tuples — delete tuple
func (h *HTTPHandler) handleReBACTuples(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.rebacWriteTuple(w, r)
	case http.MethodGet:
		h.rebacListTuples(w, r)
	case http.MethodDelete:
		h.rebacDeleteTuple(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleReBACCheck checks a permission.
// POST /api/v1/identity/check
func (h *HTTPHandler) handleReBACCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.rebacCheck(w, r)
}

func (h *HTTPHandler) rebacCheck(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.TenantID = tc.TenantID

	if req.Namespace == "" || req.Object == "" || req.Relation == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "namespace, object, relation, and subject are required")
		return
	}

	if h.rebacRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "ReBAC not configured")
		return
	}

	resp := h.rebacRepo.Check(r.Context(), req)
	writeJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) rebacWriteTuple(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Object    string `json:"object"`
		Relation  string `json:"relation"`
		Subject   string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Namespace == "" || req.Object == "" || req.Relation == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "namespace, object, relation, and subject are required")
		return
	}

	tuple := &RelationTuple{
		TenantID:  tc.TenantID,
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
		Subject:   req.Subject,
	}

	if err := h.rebacRepo.WriteTuple(r.Context(), tuple); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write tuple")
		return
	}

	writeJSON(w, http.StatusCreated, tuple)
}

func (h *HTTPHandler) rebacListTuples(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	ns := r.URL.Query().Get("namespace")
	obj := r.URL.Query().Get("object")
	rel := r.URL.Query().Get("relation")
	subj := r.URL.Query().Get("subject")

	tuples, err := h.rebacRepo.ReadTuples(r.Context(), tc.TenantID, ns, obj, rel, subj)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tuples")
		return
	}
	if tuples == nil {
		tuples = []*RelationTuple{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tuples": tuples, "total": len(tuples)})
}

func (h *HTTPHandler) rebacDeleteTuple(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Object    string `json:"object"`
		Relation  string `json:"relation"`
		Subject   string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.rebacRepo.DeleteTuple(r.Context(), tc.TenantID, req.Namespace, req.Object, req.Relation, req.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete tuple")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// SetReBACRepo injects the ReBAC tuple repository.
func (h *HTTPHandler) SetReBACRepo(repo *relationTupleRepo) {
	h.rebacRepo = repo
}

// suppress unused import warning
var _ = uuid.Nil
