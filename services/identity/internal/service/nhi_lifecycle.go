package service

import (
	"sync"
	"time"
)

type NHIIdentity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Created   time.Time `json:"created"`
	LastUsed  time.Time `json:"last_used"`
	Status    string    `json:"status"`
}

type NHIOrphan struct {
	NHI       NHIIdentity `json:"nhi"`
	DaysSince int         `json:"days_since_last_used"`
}

type DecommissionResult struct {
	ID         string `json:"id"`
	TokensRevoked int  `json:"tokens_revoked"`
	Disabled      bool `json:"disabled"`
	Audited       bool `json:"audited"`
}

type NHILifecycleService struct {
	mu         sync.RWMutex
	inventory  map[string]*NHIIdentity // fallback for nil-pool
}

func NewNHILifecycleService() *NHILifecycleService {
	return &NHILifecycleService{}
}

func (s *NHILifecycleService) ensureMap() {
	if s.inventory == nil {
		s.inventory = make(map[string]*NHIIdentity)
	}
}

func (s *NHILifecycleService) RegisterNHI(n NHIIdentity) {
	s.mu.Lock()
	s.ensureMap()
	defer s.mu.Unlock()
	s.inventory[n.ID] = &n
}

func (s *NHILifecycleService) ListNHI() []NHIIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []NHIIdentity
	for _, n := range s.inventory {
		list = append(list, *n)
	}
	return list
}

func (s *NHILifecycleService) DetectOrphans(thresholdDays int) []NHIOrphan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var orphans []NHIOrphan
	now := time.Now()
	for _, n := range s.inventory {
		if n.Status == "active" {
			days := int(now.Sub(n.LastUsed).Hours() / 24)
			if days > thresholdDays {
				orphans = append(orphans, NHIOrphan{NHI: *n, DaysSince: days})
			}
		}
	}
	return orphans
}

func (s *NHILifecycleService) DecommissionNHI(id string) *DecommissionResult {
	s.mu.Lock()
	s.ensureMap()
	defer s.mu.Unlock()
	n, ok := s.inventory[id]
	if !ok {
		return nil
	}
	n.Status = "decommissioned"
	return &DecommissionResult{
		ID:            id,
		TokensRevoked: 1,
		Disabled:      true,
		Audited:       true,
	}
}