// Package webhooks implements webhook registration, management, and delivery.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// Store is the interface for webhook persistence.
type Store interface {
	Create(ctx context.Context, wh *Webhook) error
	List(ctx context.Context, tenantID string) ([]*Webhook, error)
	Delete(ctx context.Context, id, tenantID string) error
	Get(ctx context.Context, id string) (*Webhook, error)
	ListByEvent(ctx context.Context, event string) ([]*Webhook, error)
}

// MemoryStore is an in-memory webhook store for testing/development.
type MemoryStore struct {
	mu       sync.RWMutex
	webhooks map[string]*Webhook
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{webhooks: make(map[string]*Webhook)}
}

func (s *MemoryStore) Create(_ context.Context, wh *Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhooks[wh.ID] = wh
	return nil
}

func (s *MemoryStore) List(_ context.Context, tenantID string) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Webhook
	for _, wh := range s.webhooks {
		if wh.TenantID == tenantID {
			result = append(result, wh)
		}
	}
	return result, nil
}

func (s *MemoryStore) Delete(_ context.Context, id, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	wh, ok := s.webhooks[id]
	if !ok || wh.TenantID != tenantID {
		return fmt.Errorf("not found")
	}
	delete(s.webhooks, id)
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wh, ok := s.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return wh, nil
}

func (s *MemoryStore) ListByEvent(_ context.Context, event string) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Webhook
	for _, wh := range s.webhooks {
		if !wh.Active {
			continue
		}
		for _, e := range wh.Events {
			if e == event || e == "*" {
				result = append(result, wh)
				break
			}
		}
	}
	return result, nil
}

// Handler provides HTTP handlers for webhook management.
type Handler struct {
	store     Store
	deliverer Deliverer
}

// Deliverer sends webhook payloads to external URLs.
type Deliverer interface {
	Deliver(ctx context.Context, url, secret string, payload []byte) error
}

// HTTPDeliverer delivers webhooks via HTTP POST with HMAC-SHA256 signatures.
type HTTPDeliverer struct {
	client  *http.Client
	maxRetries int
}

func NewHTTPDeliverer() *HTTPDeliverer {
	return &HTTPDeliverer{
		client:     &http.Client{Timeout: 10 * time.Second},
		maxRetries: 3,
	}
}

func (d *HTTPDeliverer) Deliver(ctx context.Context, url, secret string, payload []byte) error {
	var lastErr error
	for attempt := 0; attempt < d.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*attempt) * time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		// HMAC-SHA256 signature
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(payload)
			sig := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-GGID-Signature", "sha256="+sig)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("webhook delivery attempt %d failed: %v", attempt+1, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return lastErr
}

// NewHandler creates a webhook handler.
func NewHandler(store Store, deliverer Deliverer) *Handler {
	return &Handler{store: store, deliverer: deliverer}
}

// Create handles POST /api/v1/webhooks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, 400, map[string]string{"error": "tenant_id required"})
		return
	}

	var req struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.URL == "" || len(req.Events) == 0 {
		writeJSON(w, 400, map[string]string{"error": "url and events required"})
		return
	}

	wh := &Webhook{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		URL:       req.URL,
		Events:    req.Events,
		Secret:    req.Secret,
		CreatedAt: time.Now(),
		Active:    true,
	}
	if err := h.store.Create(r.Context(), wh); err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to create webhook"})
		return
	}
	wh.Secret = "" // don't return secret
	writeJSON(w, 201, wh)
}

// List handles GET /api/v1/webhooks
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, 400, map[string]string{"error": "tenant_id required"})
		return
	}

	webhooks, err := h.store.List(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to list webhooks"})
		return
	}
	writeJSON(w, 200, map[string]any{"webhooks": webhooks, "total": len(webhooks)})
}

// Delete handles DELETE /api/v1/webhooks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	id := r.URL.Path[len("/api/v1/webhooks/"):]
	if id == "" || tenantID == "" {
		writeJSON(w, 400, map[string]string{"error": "webhook id and tenant_id required"})
		return
	}

	if err := h.store.Delete(r.Context(), id, tenantID); err != nil {
		writeJSON(w, 404, map[string]string{"error": "webhook not found"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// Test handles POST /api/v1/webhooks/{id}/test
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	path := r.URL.Path
	// Extract ID: /api/v1/webhooks/{id}/test
	idStart := len("/api/v1/webhooks/")
	idEnd := len(path) - len("/test")
	if idEnd <= idStart || tenantID == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	id := path[idStart:idEnd]

	wh, err := h.store.Get(r.Context(), id)
	if err != nil || wh.TenantID != tenantID {
		writeJSON(w, 404, map[string]string{"error": "webhook not found"})
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"event":     "webhook.test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      map[string]string{"message": "test webhook delivery"},
	})

	err = h.deliverer.Deliver(r.Context(), wh.URL, wh.Secret, payload)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": fmt.Sprintf("delivery failed: %v", err)})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "delivered"})
}

// DeliverEvent matches an event to registered webhooks and delivers to all matches.
func (h *Handler) DeliverEvent(ctx context.Context, event string, payload []byte) {
	webhooks, err := h.store.ListByEvent(ctx, event)
	if err != nil {
		return
	}
	for _, wh := range webhooks {
		go func(w *Webhook) {
			if err := h.deliverer.Deliver(ctx, w.URL, w.Secret, payload); err != nil {
				log.Printf("webhook %s delivery failed for event %s: %v", w.ID, event, err)
			}
		}(wh)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

var _ io.Reader = (*bytes.Reader)(nil)
