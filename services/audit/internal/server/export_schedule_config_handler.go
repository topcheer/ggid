package httpserver

import (
	"encoding/json"
	"net/http"
)

type ExportJob struct {
	Name       string `json:"name"`
	Cron       string `json:"cron"`
	Format     string `json:"format"`
	Filters    string `json:"filters"`
	Retention  int    `json:"retention_days"`
	Destination string `json:"destination"`
}

type ExportScheduleConfig struct {
	Jobs          []ExportJob `json:"jobs"`
	MaxConcurrent int         `json:"max_concurrent"`
	RetryPolicy   struct {
		MaxRetries  int `json:"max_retries"`
		BackoffMins int `json:"backoff_minutes"`
	} `json:"retry_policy"`
	Notification struct {
		OnSuccess bool   `json:"on_success"`
		OnFailure bool   `json:"on_failure"`
		Channel   string `json:"channel"`
	} `json:"notification"`
}

func (s *HTTPServer) handleExportScheduleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := ExportScheduleConfig{
			Jobs: []ExportJob{
				{Name: "weekly_audit_export", Cron: "0 2 * * 0", Format: "csv", Filters: "type=security", Retention: 90, Destination: "s3://audit-bucket/weekly/"},
				{Name: "monthly_compliance", Cron: "0 0 1 * *", Format: "parquet", Filters: "type=compliance", Retention: 365, Destination: "s3://compliance/monthly/"},
				{Name: "daily_auth_log", Cron: "0 1 * * *", Format: "json", Filters: "type=auth", Retention: 30, Destination: "https://siem.example.com/ingest"},
			},
			MaxConcurrent: 3,
		}
		result.RetryPolicy.MaxRetries = 3
		result.RetryPolicy.BackoffMins = 5
		result.Notification.OnSuccess = false
		result.Notification.OnFailure = true
		result.Notification.Channel = "webhook:ops-alerts"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req ExportScheduleConfig
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
