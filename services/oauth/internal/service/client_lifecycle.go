package service

import (
	"fmt"
	"sync"
	"time"
)

type ClientStatus string

const (
	ClientActive      ClientStatus = "active"
	ClientInactive    ClientStatus = "inactive"
	ClientDeactivated ClientStatus = "deactivated"
)

type ClientLifecycle struct {
	ClientID   string       `json:"client_id"`
	Status     ClientStatus `json:"status"`
	Created    time.Time    `json:"created"`
	LastUsed   time.Time    `json:"last_used"`
	GrantTypes []string     `json:"grant_types"`
	Scopes     []string     `json:"scopes"`
	Metadata   map[string]any `json:"metadata"`
}

type ClientFilter struct {
	Status string `json:"status,omitempty"`
}

type ClientLifecycleService struct {
	mu      sync.RWMutex
	clients map[string]*ClientLifecycle
	seq     int
}

func NewClientLifecycleService() *ClientLifecycleService {
	return &ClientLifecycleService{clients: make(map[string]*ClientLifecycle)}
}

func (s *ClientLifecycleService) RegisterClient(metadata map[string]any, grantTypes, scopes []string) (*ClientLifecycle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("client_%d", s.seq)
	c := &ClientLifecycle{
		ClientID:   id,
		Status:     ClientActive,
		Created:    time.Now(),
		LastUsed:   time.Now(),
		GrantTypes: grantTypes,
		Scopes:     scopes,
		Metadata:   metadata,
	}
	s.clients[id] = c
	return c, nil
}

func (s *ClientLifecycleService) UpdateClient(id string, metadata map[string]any) (*ClientLifecycle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return nil, fmt.Errorf("client not found")
	}
	if c.Status == ClientDeactivated {
		return nil, fmt.Errorf("cannot update deactivated client")
	}
	for k, v := range metadata {
		c.Metadata[k] = v
	}
	if gt, ok := metadata["grant_types"].([]string); ok {
		c.GrantTypes = gt
	}
	if sc, ok := metadata["scopes"].([]string); ok {
		c.Scopes = sc
	}
	return c, nil
}

func (s *ClientLifecycleService) DeactivateClient(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return fmt.Errorf("client not found")
	}
	c.Status = ClientDeactivated
	return nil
}

func (s *ClientLifecycleService) DeleteClient(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[id]; !ok {
		return fmt.Errorf("client not found")
	}
	delete(s.clients, id)
	return nil
}

func (s *ClientLifecycleService) GetClientStatus(id string) *ClientLifecycle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[id]
}

func (s *ClientLifecycleService) ListClients(filter ClientFilter) []*ClientLifecycle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*ClientLifecycle
	for _, c := range s.clients {
		if filter.Status != "" && string(c.Status) != filter.Status {
			continue
		}
		list = append(list, c)
	}
	return list
}