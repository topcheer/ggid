package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Endpoint represents a configured webhook destination.
type Endpoint struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`       // event types to subscribe to
	Secret     string   `json:"secret,omitempty"` // HMAC signing secret
	MaxRetries int      `json:"max_retries"`  // default 5
	BatchSize  int      `json:"batch_size"`   // max events per delivery (default 100)
	Enabled    bool     `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

// Delivery records a single webhook delivery attempt.
type Delivery struct {
	ID           string          `json:"id"`
	EndpointID   string          `json:"endpoint_id"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"` // pending | delivered | failed | dead_letter
	Attempts     int             `json:"attempts"`
	ResponseCode int             `json:"response_code,omitempty"`
	NextRetryAt  *time.Time      `json:"next_retry_at,omitempty"`
	DeliveredAt  *time.Time      `json:"delivered_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Engine manages webhook delivery with retry, signing, and dead letter.
type Engine struct {
	pool   *pgxpool.Pool
	client *http.Client
	mu     sync.RWMutex
	// In-memory endpoint registry (for nil-pool dev/test)
	endpoints map[string]*Endpoint
}

// NewEngine creates a webhook delivery engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:      pool,
		client:    &http.Client{Timeout: 15 * time.Second},
		endpoints: make(map[string]*Endpoint),
	}
}

// EnsureSchema creates webhook tables.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webhook_endpoints (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			events TEXT[] NOT NULL DEFAULT '{}',
			secret TEXT,
			max_retries INT NOT NULL DEFAULT 5,
			batch_size INT NOT NULL DEFAULT 100,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id TEXT PRIMARY KEY,
			endpoint_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INT NOT NULL DEFAULT 0,
			response_code INT,
			next_retry_at TIMESTAMPTZ,
			delivered_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_del_ep ON webhook_deliveries(endpoint_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_webhook_del_status ON webhook_deliveries(status, next_retry_at) WHERE status = 'pending';
	`)
	return err
}

// CreateEndpoint registers a new webhook endpoint.
func (e *Engine) CreateEndpoint(ep *Endpoint) *Endpoint {
	if ep.ID == "" {
		ep.ID = uuid.New().String()
	}
	if ep.MaxRetries <= 0 {
		ep.MaxRetries = 5
	}
	if ep.BatchSize <= 0 {
		ep.BatchSize = 100
	}
	ep.CreatedAt = time.Now()
	e.mu.Lock()
	e.endpoints[ep.ID] = ep
	e.mu.Unlock()

	if e.pool != nil {
		eventsJSON, _ := json.Marshal(ep.Events)
		e.pool.Exec(context.Background(),
			`INSERT INTO webhook_endpoints (id, url, events, secret, max_retries, batch_size, enabled, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			ep.ID, ep.URL, eventsJSON, ep.Secret, ep.MaxRetries, ep.BatchSize, ep.Enabled, ep.CreatedAt)
	}
	return ep
}

// ListEndpoints returns all registered endpoints.
func (e *Engine) ListEndpoints() []*Endpoint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []*Endpoint
	for _, ep := range e.endpoints {
		result = append(result, ep)
	}
	return result
}

// GetEndpoint returns a single endpoint by ID.
func (e *Engine) GetEndpoint(id string) *Endpoint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.endpoints[id]
}

// DeleteEndpoint removes an endpoint.
func (e *Engine) DeleteEndpoint(id string) {
	e.mu.Lock()
	delete(e.endpoints, id)
	e.mu.Unlock()
	if e.pool != nil {
		e.pool.Exec(context.Background(), `DELETE FROM webhook_endpoints WHERE id = $1`, id)
	}
}

// IsSubscribed checks if an endpoint subscribes to an event type.
func (e *Engine) IsSubscribed(ep *Endpoint, eventType string) bool {
	if !ep.Enabled {
		return false
	}
	for _, ev := range ep.Events {
		if ev == eventType || ev == "*" {
			return true
		}
		// Wildcard prefix matching: "user.*" matches "user.created"
		if len(ev) > 2 && ev[len(ev)-1] == '*' {
			prefix := ev[:len(ev)-1]
			if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// Send delivers an event to all subscribed endpoints.
func (e *Engine) Send(ctx context.Context, eventType string, payload any) []*Delivery {
	payloadBytes, _ := json.Marshal(payload)

	e.mu.RLock()
	var endpoints []*Endpoint
	for _, ep := range e.endpoints {
		if e.IsSubscribed(ep, eventType) {
			endpoints = append(endpoints, ep)
		}
	}
	e.mu.RUnlock()

	var deliveries []*Delivery
	for _, ep := range endpoints {
		delivery := e.deliver(ctx, ep, eventType, payloadBytes)
		deliveries = append(deliveries, delivery)
	}
	return deliveries
}

// deliver sends a single webhook with retry logic.
func (e *Engine) deliver(ctx context.Context, ep *Endpoint, eventType string, payload []byte) *Delivery {
	d := &Delivery{
		ID:         uuid.New().String(),
		EndpointID: ep.ID,
		EventType:  eventType,
		Payload:    payload,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	maxRetries := ep.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		d.Attempts = attempt + 1
		code, err := e.sendHTTP(ctx, ep, eventType, payload)

		if err == nil && code >= 200 && code < 300 {
			d.Status = "delivered"
			d.ResponseCode = code
			now := time.Now()
			d.DeliveredAt = &now
			break
		}

		d.ResponseCode = code
		if attempt < maxRetries {
			// Exponential backoff: 1s → 2s → 4s → 8s → 16s
			backoff := time.Duration(1<<attempt) * time.Second
			nextRetry := time.Now().Add(backoff)
			d.NextRetryAt = &nextRetry
			d.Status = "pending"
			select {
			case <-ctx.Done():
				d.Status = "failed"
				return d
			case <-time.After(backoff):
			}
		} else {
			// All retries exhausted → dead letter.
			d.Status = "dead_letter"
			d.NextRetryAt = nil
		}
	}

	e.persistDelivery(ctx, d)
	return d
}

// sendHTTP sends a single HTTP POST with HMAC signature.
func (e *Engine) sendHTTP(ctx context.Context, ep *Endpoint, eventType string, payload []byte) (int, error) {
	// Wrap payload in event envelope.
	envelope := map[string]any{
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"data":       json.RawMessage(payload),
	}
	body, _ := json.Marshal(envelope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GGID-Event", eventType)

	// HMAC-SHA256 signature.
	if ep.Secret != "" {
		mac := hmac.New(sha256.New, []byte(ep.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-GGID-Signature", sig)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// GetDeliveries returns delivery history.
func (e *Engine) GetDeliveries(ctx context.Context, endpointID string, limit int) ([]Delivery, error) {
	if e.pool == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, endpoint_id, event_type, payload, status, attempts, response_code, next_retry_at, delivered_at, created_at FROM webhook_deliveries`
	args := []any{}
	if endpointID != "" {
		q += ` WHERE endpoint_id = $1`
		args = append(args, endpointID)
	}
	q += ` ORDER BY created_at DESC LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := e.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.EventType, &d.Payload, &d.Status, &d.Attempts, &d.ResponseCode, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt); err != nil {
			continue
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}

// Replay re-attempts a dead-lettered delivery.
func (e *Engine) Replay(ctx context.Context, deliveryID string) (*Delivery, error) {
	if e.pool == nil {
		return nil, fmt.Errorf("no database — cannot replay")
	}
	var d Delivery
	var payloadBytes []byte
	err := e.pool.QueryRow(ctx,
		`SELECT id, endpoint_id, event_type, payload, status FROM webhook_deliveries WHERE id = $1`, deliveryID).
		Scan(&d.ID, &d.EndpointID, &d.EventType, &payloadBytes, &d.Status)
	if err != nil {
		return nil, fmt.Errorf("delivery not found: %w", err)
	}

	ep := e.GetEndpoint(d.EndpointID)
	if ep == nil {
		return nil, fmt.Errorf("endpoint %s not found", d.EndpointID)
	}

	// Reset and retry.
	d.Payload = payloadBytes
	d.Status = "pending"
	d.Attempts = 0
	d.NextRetryAt = nil

	code, err := e.sendHTTP(ctx, ep, d.EventType, payloadBytes)
	if err == nil && code >= 200 && code < 300 {
		d.Status = "delivered"
		d.ResponseCode = code
		now := time.Now()
		d.DeliveredAt = &now
	} else {
		d.Status = "failed"
		d.ResponseCode = code
	}

	d.ID = uuid.New().String() // new delivery record for replay
	d.CreatedAt = time.Now()
	e.persistDelivery(ctx, &d)
	return &d, nil
}

// VerifySignature verifies the HMAC-SHA256 signature of a webhook payload.
func VerifySignature(payload []byte, secret, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (e *Engine) persistDelivery(ctx context.Context, d *Delivery) {
	if e.pool == nil {
		return
	}
	_, err := e.pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, endpoint_id, event_type, payload, status, attempts, response_code, next_retry_at, delivered_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		d.ID, d.EndpointID, d.EventType, d.Payload, d.Status, d.Attempts, d.ResponseCode, d.NextRetryAt, d.DeliveredAt, d.CreatedAt)
	if err != nil {
		slog.Warn("webhook delivery persist failed", "error", err)
	}
}
