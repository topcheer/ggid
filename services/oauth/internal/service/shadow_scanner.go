package service

import (
	"sync"
	"time"
)

type ShadowReport struct {
	TokenCount    int       `json:"token_count"`
	UnknownAgents []string  `json:"unknown_agents"`
	FirstSeen     time.Time `json:"first_seen"`
	LastActive    time.Time `json:"last_active"`
}

type ShadowScanner struct {
	mu             sync.RWMutex
	registeredAgents map[string]bool
}

func NewShadowScanner(registeredAgentIDs []string) *ShadowScanner {
	ra := make(map[string]bool)
	for _, id := range registeredAgentIDs {
		ra[id] = true
	}
	return &ShadowScanner{registeredAgents: ra}
}

func (s *ShadowScanner) ScanShadows(activeTokens []TokenRecord) *ShadowReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var unknown []string
	var firstSeen, lastActive time.Time

	for _, tok := range activeTokens {
		if !s.registeredAgents[tok.AgentID] {
			unknown = append(unknown, tok.AgentID)
			if firstSeen.IsZero() || tok.CreatedAt.Before(firstSeen) {
				firstSeen = tok.CreatedAt
			}
			if tok.LastUsed.After(lastActive) {
				lastActive = tok.LastUsed
			}
		}
	}

	return &ShadowReport{
		TokenCount:    len(unknown),
		UnknownAgents: unknown,
		FirstSeen:     firstSeen,
		LastActive:    lastActive,
	}
}

type TokenRecord struct {
	AgentID   string
	TokenID   string
	CreatedAt time.Time
	LastUsed  time.Time
}