package service

import (
	"fmt"
	"sync"
	"time"
)

type ProvisioningSource string

const (
	SourceHR   ProvisioningSource = "hr"
	SourceSCIM ProvisioningSource = "scim"
	SourceIaC  ProvisioningSource = "iac"
)

type ProvisioningRule struct {
	Source        ProvisioningSource `json:"source"`
	Trigger       string             `json:"trigger"`
	FieldMapping  map[string]string  `json:"field_mapping"`
	DefaultValues map[string]any     `json:"default_values"`
}

type ProvisionedUser struct {
	UserID    string              `json:"user_id"`
	Source    ProvisioningSource  `json:"source"`
	Data      map[string]any      `json:"data"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type ProvisioningAuditEntry struct {
	Action    string             `json:"action"`
	UserID    string             `json:"user_id"`
	Source    ProvisioningSource `json:"source"`
	Reason    string             `json:"reason"`
	Timestamp time.Time          `json:"timestamp"`
}

type UserProvisioningService struct {
	mu       sync.RWMutex
	users    map[string]*ProvisionedUser
	rules    map[ProvisioningSource]*ProvisioningRule
	audit    []ProvisioningAuditEntry
	seq      int
}

func NewUserProvisioningService() *UserProvisioningService {
	return &UserProvisioningService{
		users: make(map[string]*ProvisionedUser),
		rules: make(map[ProvisioningSource]*ProvisioningRule),
	}
}

func (s *UserProvisioningService) SetRule(rule ProvisioningRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.Source] = &rule
}

func (s *UserProvisioningService) ProvisionUser(source ProvisioningSource, userData map[string]any) (*ProvisionedUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	userID := fmt.Sprintf("user_%d", s.seq)
	mapped := make(map[string]any)
	rule, hasRule := s.rules[source]
	if hasRule {
		for extField, intField := range rule.FieldMapping {
			if val, ok := userData[extField]; ok {
				mapped[intField] = val
			}
		}
		for k, v := range rule.DefaultValues {
			if _, exists := mapped[k]; !exists {
				mapped[k] = v
			}
		}
	} else {
		mapped = userData
	}
	user := &ProvisionedUser{
		UserID:    userID,
		Source:    source,
		Data:      mapped,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.users[userID] = user
	s.audit = append(s.audit, ProvisioningAuditEntry{
		Action: "provision", UserID: userID, Source: source, Timestamp: time.Now(),
	})
	return user, nil
}

func (s *UserProvisioningService) DeprovisionUser(userID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.Status = "deprovisioned"
	user.UpdatedAt = time.Now()
	s.audit = append(s.audit, ProvisioningAuditEntry{
		Action: "deprovision", UserID: userID, Source: user.Source, Reason: reason, Timestamp: time.Now(),
	})
	return nil
}

func (s *UserProvisioningService) SyncUser(userID string) (*ProvisionedUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	s.audit = append(s.audit, ProvisioningAuditEntry{
		Action: "sync", UserID: userID, Source: user.Source, Timestamp: time.Now(),
	})
	return user, nil
}

func (s *UserProvisioningService) GetAuditTrail() []ProvisioningAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.audit
}