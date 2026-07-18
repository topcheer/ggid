package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/service"
)

var nhiSvc = service.NewNHILifecycleService()

func (h *HTTPHandler) handleNHIInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	nhiType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	all := nhiSvc.ListNHI()
	var filtered []service.NHIIdentity
	for _, n := range all {
		if nhiType != "" && n.Type != nhiType {
			continue
		}
		if status != "" && n.Status != status {
			continue
		}
		filtered = append(filtered, n)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (h *HTTPHandler) handleNHIOrphans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	orphans := nhiSvc.DetectOrphans(90)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orphans)
}

func (h *HTTPHandler) handleNHIDecommission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeJSONError(w, http.StatusBadRequest, "nhi id required")
		return
	}
	id := parts[len(parts)-1]
	result := nhiSvc.DecommissionNHI(id)
	if result == nil {
		writeJSONError(w, http.StatusNotFound, "nhi not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}