package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/pii"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/webhook"
	"github.com/google/uuid"
)

// AuditRepo provides audit event persistence and queries.
// Satisfied by *repository.AuditRepository.
type AuditRepo interface {
	Insert(ctx context.Context, e *domain.AuditEvent) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AuditEvent, error)
	List(ctx context.Context, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error)
	GetStats(ctx context.Context, tenantID uuid.UUID, since time.Time) (*domain.Stats, error)
	DeleteOlderThan(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error)
	// GetBoundaryEventID returns the newest event ID older than 'before' for
	// hash chain continuity preservation during retention cleanup.
	GetBoundaryEventID(ctx context.Context, tenantID uuid.UUID, before time.Time) (uuid.UUID, error)
	// DeleteOlderThanExcept deletes events older than 'before' except the event
	// with the given boundaryID (preserved as chain anchor).
	DeleteOlderThanExcept(ctx context.Context, tenantID uuid.UUID, before time.Time, exceptID uuid.UUID) (int64, error)
}

// hashChainShardCount is the number of lock shards for the per-tenant hash
// chain state. A single global lock serialized event ingestion across ALL
// tenants; sharding by tenant restores cross-tenant parallelism (P1-4).
const hashChainShardCount = 64

type hashChainShard struct {
	mu       sync.Mutex
	prevHash map[uuid.UUID]string // per-tenant last hash for chain
}

// AuditService handles audit event queries.
type AuditService struct {
	repo          AuditRepo
	shards        [hashChainShardCount]hashChainShard
	webhookEngine *webhook.Engine // optional webhook delivery engine
}

func NewAuditService(repo AuditRepo) *AuditService {
	s := &AuditService{repo: repo}
	for i := range s.shards {
		s.shards[i].prevHash = make(map[uuid.UUID]string)
	}
	return s
}

// shardFor returns the lock shard owning the given tenant's chain state.
// UUIDs are random, so the first byte distributes tenants evenly.
func (s *AuditService) shardFor(tenantID uuid.UUID) *hashChainShard {
	return &s.shards[int(tenantID[0])&(hashChainShardCount-1)]
}

// SetWebhookEngine injects the webhook delivery engine. When set, audit
// events are asynchronously delivered to registered webhooks.
func (s *AuditService) SetWebhookEngine(engine *webhook.Engine) {
	s.webhookEngine = engine
}

// GetEvent retrieves a single audit event.
func (s *AuditService) GetEvent(ctx context.Context, tenantID, id uuid.UUID) (*domain.AuditEvent, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// ListEvents returns audit events matching the filter with pagination.
// Returns (events, total, error).
func (s *AuditService) ListEvents(ctx context.Context, filter domain.ListFilter, page, pageSize int) ([]*domain.AuditEvent, int, error) {
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if filter.TenantID == uuid.Nil {
		return nil, 0, errors.InvalidArgument("tenant_id is required")
	}
	events, total, err := s.repo.List(ctx, filter, pageSize, offset)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrInternal, "list audit events", err)
	}
	return events, total, nil
}

// InsertEvent directly inserts an audit event (for testing or synchronous use).
// PII fields (email, phone, IP, SSN) in ActorName, ResourceName, and Metadata
// are obfuscated before persistence. A SHA-256 hash chain is computed:
// hash = SHA256(prev_hash + canonical_event_data).
func (s *AuditService) InsertEvent(ctx context.Context, event *domain.AuditEvent) error {
	ObfuscateEventPII(event)
	// Compute an initial hash using in-memory chain state (for tests/mocks).
	// repo.Insert will override with the authoritative DB-backed hash using
	// a FOR UPDATE transaction to prevent race conditions.
	s.computeHashChain(event)
	if err := s.repo.Insert(ctx, event); err != nil {
		return errors.Wrap(errors.ErrInternal, "insert audit event", err)
	}

	// Asynchronously deliver to registered webhooks (best-effort, non-blocking).
	if s.webhookEngine != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("webhook delivery panic recovered", "error", r)
				}
			}()
			s.webhookEngine.Send(context.Background(), event.Action, event)
		}()
	}

	return nil
}

// obfuscateEventPII masks PII fields in an audit event before storage.
// This prevents raw emails, phone numbers, and other sensitive data from
// being persisted in the audit log.
// ObfuscateEventPII masks PII fields in an audit event before storage.
func ObfuscateEventPII(e *domain.AuditEvent) {
	e.ActorName = pii.Obfuscate(e.ActorName)
	e.ResourceName = pii.Obfuscate(e.ResourceName)
	e.IPAddress = pii.MaskIP(e.IPAddress)
	if e.Metadata != nil {
		for k, v := range e.Metadata {
			if s, ok := v.(string); ok {
				e.Metadata[k] = pii.Obfuscate(s)
			} else {
				// Marshal/unmarshal to mask nested string values
				if raw, err := json.Marshal(v); err == nil {
					masked := pii.Obfuscate(string(raw))
					var nv any
					if json.Unmarshal([]byte(masked), &nv) == nil {
						e.Metadata[k] = nv
					}
				}
			}
		}
	}
}

// GetStats returns aggregated audit analytics for the last 24 hours.
func (s *AuditService) GetStats(ctx context.Context, tenantID uuid.UUID) (*domain.Stats, error) {
	if tenantID == uuid.Nil {
		return nil, errors.InvalidArgument("tenant_id is required")
	}
	since := time.Now().UTC().Add(-24 * time.Hour)
	return s.repo.GetStats(ctx, tenantID, since)
}

// CleanupOldEvents deletes audit events older than the retention period.
// SECURITY: preserves hash chain integrity by keeping the newest event within
// the deletion window — its prev_hash links to the deleted range, so the chain
// can still be verified from that point forward.
// Returns the number of deleted events.
func (s *AuditService) CleanupOldEvents(ctx context.Context, tenantID uuid.UUID, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // default 90 days
	}
	before := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	// Preserve chain continuity: keep the newest event at or before the cutoff.
	// This event's prev_hash references the deleted range, serving as the chain
	// boundary anchor for verification of all subsequent events.
	boundaryID, err := s.repo.GetBoundaryEventID(ctx, tenantID, before)
	if err != nil || boundaryID == uuid.Nil {
		// No events in deletion window — nothing to clean.
		return 0, nil
	}
	return s.repo.DeleteOlderThanExcept(ctx, tenantID, before, boundaryID)
}

// computeHashChain delegates to the domain layer's ComputeHash which uses
// HMAC-SHA256 with length-prefix canonicalization (P2-6/P2-7). The old
// inline implementation used plain SHA256 with pipe-delimiters — a different
// hash that tamper-check (domain.VerifyHash) could never verify.
func (s *AuditService) computeHashChain(event *domain.AuditEvent) {
	if !domain.IsHashChainEnabled() {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	// Set PrevHash from the in-memory chain state for this tenant.
	// First event for a tenant gets empty PrevHash (genesis).
	// Only this tenant's shard is locked — other tenants ingest in parallel.
	sh := s.shardFor(event.TenantID)
	sh.mu.Lock()
	event.PrevHash = sh.prevHash[event.TenantID]
	event.Hash = event.ComputeHash(event.PrevHash)
	// Update chain state for the next event.
	sh.prevHash[event.TenantID] = event.Hash
	sh.mu.Unlock()
}

// canonicalEventData produces a deterministic byte representation of an event
// for hash chain computation. Fields are sorted for reproducibility.
func canonicalEventData(e *domain.AuditEvent) []byte {
	parts := []string{
		e.ID.String(),
		e.TenantID.String(),
		string(e.ActorType),
		e.ActorName,
		e.Action,
		e.ResourceType,
		e.ResourceName,
		string(e.Result),
		e.IPAddress,
		e.UserAgent,
		e.RequestID,
		e.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	// Include metadata keys in sorted order for determinism.
	if len(e.Metadata) > 0 {
		keys := make([]string, 0, len(e.Metadata))
		for k := range e.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if raw, err := json.Marshal(e.Metadata[k]); err == nil {
				parts = append(parts, k+"="+string(raw))
			}
		}
	}
	return []byte(strings.Join(parts, "|"))
}

// VerifyIntegrity checks that a sequence of events forms a valid hash chain.
// Events must be in chronological order. Returns nil if the chain is valid,
// or an error describing the first break detected.
func (s *AuditService) VerifyIntegrity(ctx context.Context, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return errors.InvalidArgument("tenant_id is required")
	}

	events, _, err := s.ListEvents(ctx, domain.ListFilter{
		TenantID: tenantID,
		OrderBy:  "created_at",
	}, 1, 500)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	var prevHash string
	for i, e := range events {
		if e.PrevHash != prevHash {
			return fmt.Errorf("hash chain broken at event %d (id=%s): prev_hash mismatch", i, e.ID)
		}

		if !e.VerifyHash(prevHash) {
			return fmt.Errorf("hash chain broken at event %d (id=%s): hash mismatch (tampered)", i, e.ID)
		}
		prevHash = e.Hash
	}
	return nil
}
