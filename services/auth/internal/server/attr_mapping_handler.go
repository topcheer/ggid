package server

import (
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// handleAttrMappings handles CRUD for attribute mappings.
// GET    /api/v1/admin/migration/mappings     — list all
// POST   /api/v1/admin/migration/mappings     — create
// GET    /api/v1/admin/migration/mappings/:id — get one
// PUT    /api/v1/admin/migration/mappings/:id — update
// DELETE /api/v1/admin/migration/mappings/:id — delete
// POST   /api/v1/admin/migration/mappings/test — test resolution
func (h *Handler) handleAttrMappings(w http.ResponseWriter, r *http.Request) {
	// Route by sub-path.
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/migration/mappings")
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.attrMappingList(w, r)
		case http.MethodPost:
			h.attrMappingCreate(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Sub-routes: /test or /:id
	if path == "test" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.attrMappingTest(w, r)
		return
	}

	// It's an :id path.
	mappingID := path
	switch r.Method {
	case http.MethodGet:
		h.attrMappingGet(w, r, mappingID)
	case http.MethodPut:
		h.attrMappingUpdate(w, r, mappingID)
	case http.MethodDelete:
		h.attrMappingDelete(w, r, mappingID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) attrMappingList(w http.ResponseWriter, r *http.Request) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	mappings, err := h.attrMapRepo.List(r.Context(), tc.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list mappings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mappings": mappings,
		"count":    len(mappings),
	})
}

func (h *Handler) attrMappingCreate(w http.ResponseWriter, r *http.Request) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	var m AttributeMapping
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m.TenantID = tc.TenantID.String()
	if err := ValidateMapping(&m); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.attrMapRepo.Create(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create mapping")
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) attrMappingGet(w http.ResponseWriter, r *http.Request, id string) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	m, err := h.attrMapRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "mapping not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) attrMappingUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	var m AttributeMapping
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m.ID = id
	if err := ValidateMapping(&m); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.attrMapRepo.Update(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update mapping")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) attrMappingDelete(w http.ResponseWriter, r *http.Request, id string) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	if err := h.attrMapRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete mapping")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) attrMappingTest(w http.ResponseWriter, r *http.Request) {
	if h.attrMapRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	var input map[string][]string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.attrMapRepo.TestMapping(r.Context(), tc.TenantID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "test failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
