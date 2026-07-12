package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/auth/internal/service"
)

var rotSvc = service.NewRotationScheduler()

func (h *Handler) handleRotationRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/rotation") {
		h.handleRotationSchedule(w, r)
		return
	}
	if strings.HasSuffix(path, "/rotation/execute") {
		h.handleRotationExecute(w, r)
		return
	}
	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}

func (h *Handler) handleRotationSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	id := parts[len(parts)-2]
	var policy service.RotationPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	sched := rotSvc.ScheduleRotation(id, policy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sched)
}

func (h *Handler) handleRotationDue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	due := rotSvc.CheckDueRotations()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(due)
}

func (h *Handler) handleRotationExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	id := parts[len(parts)-2]
	result := rotSvc.ExecuteRotation(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}