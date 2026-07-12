package service

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type PasswordHistoryEntry struct {
	UserID       string    `json:"user_id"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type PasswordHistoryService struct {
	mu         sync.RWMutex
	history    map[string][]PasswordHistoryEntry // userID -> entries (newest first)
	maxHistory int
}

func NewPasswordHistoryService(maxHistory int) *PasswordHistoryService {
	return &PasswordHistoryService{
		history:    make(map[string][]PasswordHistoryEntry),
		maxHistory: maxHistory,
	}
}

func (s *PasswordHistoryService) AddPasswordHistory(userID, hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := PasswordHistoryEntry{
		UserID:       userID,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}
	entries := s.history[userID]
	entries = append([]PasswordHistoryEntry{entry}, entries...)
	if len(entries) > s.maxHistory {
		entries = entries[:s.maxHistory]
	}
	s.history[userID] = entries
}

func (s *PasswordHistoryService) CheckPasswordHistory(userID, hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hashed := hashPassword(hash)
	for _, entry := range s.history[userID] {
		if entry.PasswordHash == hashed || entry.PasswordHash == hash {
			return true // duplicate found
		}
	}
	return false
}

func (s *PasswordHistoryService) GetPasswordHistory(userID string, limit int) []PasswordHistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.history[userID]
	if limit > 0 && len(entries) > limit {
		return entries[:limit]
	}
	return entries
}

func (s *PasswordHistoryService) PurgeOldEntries(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.history[userID]
	if len(entries) <= s.maxHistory {
		return 0
	}
	purged := len(entries) - s.maxHistory
	s.history[userID] = entries[:s.maxHistory]
	return purged
}

func hashPassword(pw string) string {
	h := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(h[:])
}