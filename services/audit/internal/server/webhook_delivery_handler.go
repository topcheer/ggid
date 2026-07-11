package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WebhookDelivery tracks delivery attempts for webhook events.
type WebhookDelivery struct {
	ID           string     `json:"id"`
	WebhookID    string     `json:"webhook_id"`
	EventID      string     `json:"event_id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"` // pending, delivered, failed, retrying
	Attempts     int        `json:"attempts"`
	LastError    string     `json:"last_error,omitempty"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type webhookDeliveryStore struct {
	mu        sync.RWMutex
	deliveries map[string]*WebhookDelivery
}

var webhookDeliveries = &webhookDeliveryStore{deliveries: make(map[string]*WebhookDelivery)}

// RecordWebhookDelivery adds or updates a delivery record.
func RecordWebhookDelivery(webhookID, eventID, eventType string) *WebhookDelivery {
	id := uuid.New().String()
	d := &WebhookDelivery{
		ID:        id,
		WebhookID: webhookID,
		EventID:   eventID,
		EventType: eventType,
		Status:    "pending",
		Attempts:  0,
		CreatedAt: time.Now().UTC(),
	}
	webhookDeliveries.mu.Lock()
	webhookDeliveries.deliveries[id] = d
	webhookDeliveries.mu.Unlock()
	return d
}

// GET /api/v1/audit/webhooks/delivery-status — list failed deliveries.
// POST /api/v1/audit/webhooks/{id}/retry — manual retry.
func (s *HTTPServer) handleWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	// Check for /retry sub-path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/webhooks/")
	if strings.HasSuffix(path, "/retry") {
		webhookID := strings.TrimSuffix(path, "/retry")
		s.retryWebhookDelivery(w, r, webhookID)
		return
	}

	// GET delivery-status
	if path == "delivery-status" || path == "" {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		statusFilter := r.URL.Query().Get("status")
		if statusFilter == "" {
			statusFilter = "failed" // default to showing failures
		}

		webhookDeliveries.mu.RLock()
		result := []*WebhookDelivery{}
		for _, d := range webhookDeliveries.deliveries {
			if statusFilter != "all" && d.Status != statusFilter {
				continue
			}
			result = append(result, d)
		}
		webhookDeliveries.mu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"deliveries": result,
			"count":      len(result),
		})
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) retryWebhookDelivery(w http.ResponseWriter, r *http.Request, webhookID string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	webhookDeliveries.mu.Lock()
	defer webhookDeliveries.mu.Unlock()

	// Find deliveries for this webhook
	retried := 0
	var failed []map[string]string
	for _, d := range webhookDeliveries.deliveries {
		if d.WebhookID != webhookID {
			continue
		}
		if d.Status == "delivered" {
			continue
		}
		d.Attempts++
		d.LastError = ""
		d.Status = "retrying"
		nextRetry := time.Now().UTC().Add(30 * time.Second)
		d.NextRetryAt = &nextRetry
		retried++
	}

	if retried == 0 {
		failed = append(failed, map[string]string{
			"webhook_id": webhookID,
			"error":      "no failed deliveries found for this webhook",
		})
	}

	if failed == nil {
		failed = []map[string]string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "retrying",
		"webhook_id":    webhookID,
		"retried_count": retried,
		"failed":        failed,
		"next_check":    fmt.Sprintf("%ds", 30),
	})
}
