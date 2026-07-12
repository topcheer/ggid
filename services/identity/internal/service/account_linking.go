package service

import (
	"fmt"
	"sync"
	"time"
)

type LinkedAccount struct {
	LinkID     string    `json:"link_id"`
	UserID     string    `json:"user_id"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	LinkedAt   time.Time `json:"linked_at"`
	LastSync   time.Time `json:"last_sync"`
	Status     string    `json:"status"`
}

type AccountLinkingService struct {
	mu      sync.RWMutex
	links   map[string]*LinkedAccount // linkID -> account
	byUser  map[string][]string       // userID -> []linkID
	dedupe  map[string]string         // "userID:provider:externalID" -> linkID
	seq     int
}

func NewAccountLinkingService() *AccountLinkingService {
	return &AccountLinkingService{
		links:  make(map[string]*LinkedAccount),
		byUser: make(map[string][]string),
		dedupe: make(map[string]string),
	}
}

func (s *AccountLinkingService) LinkAccount(userID, provider, externalID string) (*LinkedAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dedupeKey := fmt.Sprintf("%s:%s:%s", userID, provider, externalID)
	if existingID, exists := s.dedupe[dedupeKey]; exists {
		return nil, fmt.Errorf("account already linked: %s", existingID)
	}
	s.seq++
	link := &LinkedAccount{
		LinkID:     fmt.Sprintf("link_%d", s.seq),
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
		LinkedAt:   time.Now(),
		LastSync:   time.Now(),
		Status:     "active",
	}
	s.links[link.LinkID] = link
	s.byUser[userID] = append(s.byUser[userID], link.LinkID)
	s.dedupe[dedupeKey] = link.LinkID
	return link, nil
}

func (s *AccountLinkingService) UnlinkAccount(linkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[linkID]
	if !ok {
		return fmt.Errorf("link not found")
	}
	delete(s.links, linkID)
	dedupeKey := fmt.Sprintf("%s:%s:%s", link.UserID, link.Provider, link.ExternalID)
	delete(s.dedupe, dedupeKey)
	var filtered []string
	for _, id := range s.byUser[link.UserID] {
		if id != linkID {
			filtered = append(filtered, id)
		}
	}
	s.byUser[link.UserID] = filtered
	return nil
}

func (s *AccountLinkingService) ListLinkedAccounts(userID string) []*LinkedAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*LinkedAccount
	for _, lid := range s.byUser[userID] {
		if l, ok := s.links[lid]; ok {
			list = append(list, l)
		}
	}
	return list
}

func (s *AccountLinkingService) SyncLinkedAccount(linkID string) (*LinkedAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[linkID]
	if !ok {
		return nil, fmt.Errorf("link not found")
	}
	link.LastSync = time.Now()
	return link, nil
}