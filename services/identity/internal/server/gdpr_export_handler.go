package server

import (
	"encoding/json"
	"net/http"
)

type GDPRDataExport struct {
	ExportID    string                 `json:"export_id"`
	UserID      string                 `json:"user_id"`
	Status      string                 `json:"status"`
	Format      string                 `json:"format"`
	DataCategories []string            `json:"data_categories"`
	DownloadURL string                 `json:"download_url,omitempty"`
	ExpiresAt   string                 `json:"expires_at"`
	RequestedAt string                 `json:"requested_at"`
	Summary     map[string]int         `json:"summary"`
}

func (h *HTTPHandler) handleGDPRExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := GDPRDataExport{
		ExportID:      "export-001",
		UserID:        "u-0342",
		Status:        "processing",
		Format:        "json",
		DataCategories: []string{"profile", "auth_events", "audit_trail", "consent_records", "sessions"},
		ExpiresAt:     "2025-01-22T10:00:00Z",
		RequestedAt:   "2025-01-15T10:00:00Z",
		Summary: map[string]int{
			"profile":        1,
			"auth_events":    342,
			"audit_trail":    1850,
			"consent_records": 4,
			"sessions":       28,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(result)
}
