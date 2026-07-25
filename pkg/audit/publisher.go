// Package audit provides a lightweight NATS publisher for audit events.
// Any service can use this to publish audit events to the NATS JetStream
// audit stream, which the Audit Service consumes and persists to the database.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Default stream and subject names. These must match the Audit Service consumer.
const (
	DefaultStreamName  = "AUDIT"
	DefaultSubjectName = "audit.events"
)

// Event is the audit event message published to NATS.
// It maps directly to the AuditEvent domain model in the Audit Service.
type Event struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	ActorType    string         `json:"actor_type"`    // user | api_key | system | anonymous
	ActorID      uuid.UUID      `json:"actor_id"`
	ActorName    string         `json:"actor_name"`
	Action       string         `json:"action"`        // e.g. "user.login", "role.assign"
	ResourceType string         `json:"resource_type"`
	ResourceID   uuid.UUID      `json:"resource_id"`
	ResourceName string         `json:"resource_name"`
	Result       string         `json:"result"`        // success | failure | denied
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	RequestID    string         `json:"request_id"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Publisher publishes audit events to NATS JetStream.
type Publisher struct {
	nc      *nats.Conn
	js      jetstream.JetStream
	stream  string
	subject string
}

// NewPublisher creates a new audit event publisher connected to NATS.
// It ensures the audit stream exists (creates if missing).
func NewPublisher(ctx context.Context, natsURL string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL,
		nats.Name("ggid-audit-publisher"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	// Ensure the audit stream exists.
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     DefaultStreamName,
		Subjects: []string{DefaultSubjectName},
		Retention: jetstream.LimitsPolicy,
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure audit stream: %w", err)
	}

	return &Publisher{
		nc:      nc,
		js:      js,
		stream:  DefaultStreamName,
		subject: DefaultSubjectName,
	}, nil
}

// Publish publishes a single audit event to NATS.
// The event is marshalled to JSON and published asynchronously.
// If publishing fails, the error is returned but the event is not retried.
func (p *Publisher) Publish(ctx context.Context, event Event) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	// Sanitize IP address: strip port (e.g. "10.42.0.83:39820" → "10.42.0.83")
	// PostgreSQL inet type rejects IP:PORT format.
	event.IPAddress = sanitizeIPAddress(event.IPAddress)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	_, err = p.js.Publish(ctx, p.subject, data)
	if err != nil {
		return fmt.Errorf("publish audit event: %w", err)
	}
	return nil
}

// PublishAsync publishes an audit event. Uses synchronous Publish to ensure
// the message reaches JetStream before returning (PublishAsync was unreliable
// in practice — messages could be lost if the request goroutine exited).
func (p *Publisher) PublishAsync(event Event) {
	if p.js == nil {
		return
	}
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Use synchronous Publish with a short timeout — audit events are best-effort
	// but should at least reach JetStream.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = p.js.Publish(ctx, p.subject, data)
	if err != nil {
		// Best-effort: don't block the request on audit failures.
		return
	}
}

// Close closes the underlying NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

// sanitizeIPAddress strips the port from an IP address string.
// PostgreSQL inet type rejects "IP:PORT" format (e.g. "10.42.0.83:39820").
// For IPv6 addresses like "[::1]:8080", it extracts the address inside brackets.
func sanitizeIPAddress(ip string) string {
	if ip == "" {
		return ""
	}
	// Handle IPv6 with brackets: [::1]:8080
	if strings.HasPrefix(ip, "[") {
		if end := strings.Index(ip, "]"); end > 0 {
			return ip[1:end]
		}
	}
	// Strip port for IPv4: 10.42.0.83:39820
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		return ip[:idx]
	}
	return ip
}

// NewEvent is a convenience function to create an audit event with defaults.
func NewEvent(action string, result string, tenantID uuid.UUID, actorID uuid.UUID) Event {
	return Event{
		TenantID:  tenantID,
		ActorType: "user",
		ActorID:   actorID,
		Action:    action,
		Result:    result,
		CreatedAt: time.Now(),
	}
}

// sanitizeIPAddress strips the port from an IP address.
// PostgreSQL inet type rejects IP:PORT format (e.g. "10.42.0.83:39820" → "10.42.0.83").
func sanitizeIPAddress(ip string) string {
	if ip == "" {
		return ip
	}
	// Handle IPv6 with brackets: [::1]:12345
	if ip[0] == '[' {
		if idx := strings.LastIndex(ip, "]"); idx > 0 {
			return ip[1:idx]
		}
	}
	// Strip port from IPv4 or plain IPv6
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		// Avoid mangling bare IPv6 (e.g. "::1" has colons but no port)
		// Only strip if there's exactly one colon and it looks like IPv4:port
		if strings.Count(ip, ":") == 1 {
			return ip[:idx]
		}
	}
	return ip
}
