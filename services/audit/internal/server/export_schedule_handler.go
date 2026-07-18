package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ExportScheduleRequest struct {
	Format        string            `json:"format"`
	ScheduleCron  string            `json:"schedule_cron"`
	Filters       map[string]string `json:"filters"`
	RetentionDays int               `json:"retention_days"`
	Destination   string            `json:"destination"`
	DestConfig    map[string]string `json:"dest_config"`
}

type ExportSchedule struct {
	ScheduleID    string            `json:"schedule_id"`
	Format        string            `json:"format"`
	ScheduleCron  string            `json:"schedule_cron"`
	Filters       map[string]string `json:"filters"`
	RetentionDays int               `json:"retention_days"`
	Destination   string            `json:"destination"`
	DestConfig    map[string]string `json:"dest_config"`
	LastExportAt  string            `json:"last_export_at"`
	NextExportAt  string            `json:"next_export_at"`
	Status        string            `json:"status"`
	CreatedAt     string            `json:"created_at"`
}

var exportScheduleStore sync.Map

func (s *HTTPServer) handleExportSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ExportScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Format == "" {
		req.Format = "json"
	}
	if req.ScheduleCron == "" {
		req.ScheduleCron = "0 2 * * *"
	}
	if req.Destination == "" {
		req.Destination = "s3"
	}
	if req.RetentionDays == 0 {
		req.RetentionDays = 30
	}

	now := time.Now().UTC()
	sched := ExportSchedule{
		ScheduleID:    fmt.Sprintf("exp-%d", now.UnixNano()%100000),
		Format:        req.Format,
		ScheduleCron:  req.ScheduleCron,
		Filters:       req.Filters,
		RetentionDays: req.RetentionDays,
		Destination:   req.Destination,
		DestConfig:    req.DestConfig,
		LastExportAt:  "",
		NextExportAt:  now.Add(24 * time.Hour).Format(time.RFC3339),
		Status:        "active",
		CreatedAt:     now.Format(time.RFC3339),
	}
	exportScheduleStore.Store(sched.ScheduleID, sched)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sched)
}
