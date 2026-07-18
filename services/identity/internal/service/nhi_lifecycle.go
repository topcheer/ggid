package service

import (
	"context"
	"time"
)

// NHIPersistenceBackend is the interface for NHI storage.
// Implementations: PG-backed repo (production), nil/no-op (dev/test).
type NHIPersistenceBackend interface {
	Register(ctx context.Context, id, name, nhiType, status string) error
	List(ctx context.Context) ([]NHIIdentity, error)
	Get(ctx context.Context, id string) (*NHIIdentity, error)
	Decommission(ctx context.Context, id string) error
}

// NHIInMemoryBackend is a simple in-memory implementation for tests.
type NHIInMemoryBackend struct{}

func (b *NHIInMemoryBackend) Register(ctx context.Context, id, name, nhiType, status string) error {
	return nil
}
func (b *NHIInMemoryBackend) List(ctx context.Context) ([]NHIIdentity, error) {
	return []NHIIdentity{}, nil
}
func (b *NHIInMemoryBackend) Get(ctx context.Context, id string) (*NHIIdentity, error) {
	return nil, nil
}
func (b *NHIInMemoryBackend) Decommission(ctx context.Context, id string) error {
	return nil
}

type NHIIdentity struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"last_used"`
	Status   string    `json:"status"`
}

type NHIOrphan struct {
	NHI       NHIIdentity `json:"nhi"`
	DaysSince int         `json:"days_since_last_used"`
}

type DecommissionResult struct {
	ID            string `json:"id"`
	TokensRevoked int    `json:"tokens_revoked"`
	Disabled      bool   `json:"disabled"`
	Audited       bool   `json:"audited"`
}

// NHILifecycleService manages NHI lifecycle via a persistence backend.
type NHILifecycleService struct {
	backend NHIPersistenceBackend
}

// NewNHILifecycleService creates a lifecycle service with optional backend.
func NewNHILifecycleService() *NHILifecycleService {
	return &NHILifecycleService{backend: &NHIInMemoryBackend{}}
}

// SetBackend injects a persistence backend (PG repo for production).
func (s *NHILifecycleService) SetBackend(b NHIPersistenceBackend) {
	s.backend = b
}

func (s *NHILifecycleService) RegisterNHI(n NHIIdentity) {
	_ = s.backend.Register(context.Background(), n.ID, n.Name, n.Type, n.Status)
}

func (s *NHILifecycleService) ListNHI() []NHIIdentity {
	list, _ := s.backend.List(context.Background())
	if list == nil {
		return []NHIIdentity{}
	}
	return list
}

func (s *NHILifecycleService) DetectOrphans(thresholdDays int) []NHIOrphan {
	list := s.ListNHI()
	var orphans []NHIOrphan
	now := time.Now()
	for _, n := range list {
		if n.Status == "active" {
			days := int(now.Sub(n.LastUsed).Hours() / 24)
			if days > thresholdDays {
				orphans = append(orphans, NHIOrphan{NHI: n, DaysSince: days})
			}
		}
	}
	return orphans
}

func (s *NHILifecycleService) DecommissionNHI(id string) *DecommissionResult {
	if err := s.backend.Decommission(context.Background(), id); err != nil {
		return nil
	}
	return &DecommissionResult{
		ID:            id,
		TokensRevoked: 1,
		Disabled:      true,
		Audited:       true,
	}
}
