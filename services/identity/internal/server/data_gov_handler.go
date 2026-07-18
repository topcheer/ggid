package server

import (
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// handleDataGovernance routes data governance endpoints.
func (h *HTTPHandler) handleDataGovernance(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/data-governance/classifications"):
		h.dgClassifications(w, r)
	case strings.HasSuffix(path, "/data-governance/dsr"):
		h.dgDSR(w, r)
	case strings.HasSuffix(path, "/data-governance/inventory"):
		h.dgInventory(w, r)
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}

func (h *HTTPHandler) dgClassifications(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodPost:
		var dc DataClassification
		if err := json.NewDecoder(r.Body).Decode(&dc); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		dc.TenantID = tc.TenantID
		if dc.ResourceType == "" || dc.ResourceID == "" {
			writeJSONError(w, http.StatusBadRequest, "resource_type and resource_id required")
			return
		}
		if dc.Classification == "" {
			dc.Classification = "general"
		}
		if dc.CrossBorder == "" {
			dc.CrossBorder = "allowed"
		}
		if err := h.dataGovRepo.CreateClassification(r.Context(), &dc); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		writeJSON(w, http.StatusCreated, dc)
	case http.MethodGet:
		list, err := h.dataGovRepo.ListClassifications(r.Context(), tc.TenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		if list == nil {
			list = []*DataClassification{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"classifications": list, "total": len(list)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) dgDSR(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodPost:
		var req DSRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.TenantID = tc.TenantID
		req.Status = "pending"
		if req.RequestType == "" {
			writeJSONError(w, http.StatusBadRequest, "request_type required")
			return
		}
		if err := h.dataGovRepo.CreateDSR(r.Context(), &req); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		writeJSON(w, http.StatusCreated, req)
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		list, err := h.dataGovRepo.ListDSR(r.Context(), tc.TenantID, status)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed")
			return
		}
		if list == nil {
			list = []*DSRRequest{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": list, "total": len(list)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) dgInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}
	// Query DB-backed classifications to build inventory.
	classifications, err := h.dataGovRepo.ListClassifications(r.Context(), tc.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data_categories": classifications,
		"total":           len(classifications),
		"generated_at":    "live",
	})
}
