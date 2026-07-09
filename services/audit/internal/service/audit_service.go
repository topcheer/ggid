package service

import (
	"context"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/google/uuid"
)

// AuditService handles audit event queries.
type AuditService struct {
	repo *repository.AuditRepository
}

func NewAuditService(repo *repository.AuditRepository) *AuditService {
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
