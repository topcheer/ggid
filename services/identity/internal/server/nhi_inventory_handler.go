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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	orphans := nhiSvc.DetectOrphans(90)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orphans)
}

func (h *HTTPHandler) handleNHIDecommission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"nhi id required"}`, http.StatusBadRequest)
		return
	}
	id := parts[len(parts)-1]
	result := nhiSvc.DecommissionNHI(id)
	if result == nil {
		http.Error(w, `{"error":"nhi not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}