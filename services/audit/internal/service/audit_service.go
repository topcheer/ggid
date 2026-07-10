package service

import (
	"context"
	"time"

	"github.com/ggid/ggid/pkg/errors"
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
	repo AuditRepo
}

func NewAuditService(repo AuditRepo) *AuditService {
	return &AuditService{repo: repo}
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
func (s *AuditService) InsertEvent(ctx context.Context, event *domain.AuditEvent) error {
	if err := s.repo.Insert(ctx, event); err != nil {
		return errors.Wrap(errors.ErrInternal, "insert audit event", err)
	}
	return nil
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