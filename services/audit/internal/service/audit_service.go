package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/pii"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// AuditRepo provides audit event persistence and queries.
// Satisfied by *repository.AuditRepository.
type AuditRepo interface {
	Insert(ctx context.Context, e *domain.AuditEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	List(ctx context.Context, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error)
	GetStats(ctx context.Context, tenantID uuid.UUID, since time.Time) (*domain.Stats, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// AuditService handles audit event queries.
type AuditService struct {
	repo      AuditRepo
	mu        sync.RWMutex
	prevHash  map[uuid.UUID]string // per-tenant last hash for chain
}

func NewAuditService(repo AuditRepo) *AuditService {
	return &AuditService{repo: repo, prevHash: make(map[uuid.UUID]string)}
}

// GetEvent retrieves a single audit event.
func (s *AuditService) GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	return s.repo.GetByID(ctx, id)
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
	obfuscateEventPII(event)
	s.computeHashChain(event)
	if err := s.repo.Insert(ctx, event); err != nil {
		return errors.Wrap(errors.ErrInternal, "insert audit event", err)
	}
	return nil
}

// obfuscateEventPII masks PII fields in an audit event before storage.
// This prevents raw emails, phone numbers, and other sensitive data from
// being persisted in the audit log.
func obfuscateEventPII(e *domain.AuditEvent) {
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
// Returns the number of deleted events.
func (s *AuditService) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // default 90 days
	}
	before := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	return s.repo.DeleteOlderThan(ctx, before)
}

// computeHashChain computes the SHA-256 hash chain for an event.
// Each event's hash = SHA256(prev_hash + canonical_event_data).
// The prev_hash is tracked per-tenant in memory.
func (s *AuditService) computeHashChain(event *domain.AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.prevHash == nil {
		s.prevHash = make(map[uuid.UUID]string)
	}

	prev := s.prevHash[event.TenantID]
	event.PrevHash = prev

	data := canonicalEventData(event)
	h := sha256.Sum256(append([]byte(prev), data...))
	event.Hash = hex.EncodeToString(h[:])
	s.prevHash[event.TenantID] = event.Hash
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

		data := canonicalEventData(e)
		h := sha256.Sum256(append([]byte(prevHash), data...))
		expected := hex.EncodeToString(h[:])
		if e.Hash != expected {
			return fmt.Errorf("hash chain broken at event %d (id=%s): hash mismatch (tampered)", i, e.ID)
		}
		prevHash = e.Hash
	}
	return nil
}