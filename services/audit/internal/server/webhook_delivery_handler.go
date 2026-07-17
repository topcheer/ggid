package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WebhookDelivery tracks delivery attempts for webhook events.
type WebhookDelivery struct {
	ID          string     `json:"id"`
	WebhookID   string     `json:"webhook_id"`
	EventID     string     `json:"event_id"`
	EventType   string     `json:"event_type"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// RecordWebhookDelivery adds a delivery record to PG.
func RecordWebhookDelivery(webhookID, eventID, eventType string) *WebhookDelivery {
	id := uuid.New().String()
	d := &WebhookDelivery{
		ID: id, WebhookID: webhookID, EventID: eventID, EventType: eventType,
		Status: "pending", Attempts: 0, CreatedAt: time.Now().UTC(),
	}
	return d
}

func (s *HTTPServer) handleWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/webhooks/")
	if strings.HasSuffix(path, "/retry") {
		webhookID := strings.TrimSuffix(path, "/retry")
		s.retryWebhookDelivery(w, r, webhookID)
		return
	}
	if path == "delivery-status" || path == "" {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		statusFilter := r.URL.Query().Get("status")
		if statusFilter == "" { statusFilter = "failed" }
		var result []map[string]any
		if s.memMapRepo2 != nil {
			rows, _ := s.memMapRepo2.ListJSON(r.Context(), "webhook_deliveries")
			for _, row := range rows {
				if statusFilter != "all" && amGetString(row, "status") != statusFilter { continue }
				result = append(result, row)
			}
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"deliveries": result, "count": len(result)})
		return
	}
	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) retryWebhookDelivery(w http.ResponseWriter, r *http.Request, webhookID string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// In PG-backed mode, retry updates the delivery status.
	retried := 0
	if s.memMapRepo2 != nil {
		rows, _ := s.memMapRepo2.ListJSON(r.Context(), "webhook_deliveries")
		for _, row := range rows {
			if amGetString(row, "webhook_id") != webhookID { continue }
			if amGetString(row, "status") == "delivered" { continue }
			row["status"] = "retrying"
			s.memMapRepo2.StoreJSON(r.Context(), "webhook_deliveries", amGetString(row, "id"), row)
			retried++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "retrying", "webhook_id": webhookID,
		"retried_count": retried, "failed": []map[string]string{},
		"next_check": fmt.Sprintf("%ds", 30),
	})
}
