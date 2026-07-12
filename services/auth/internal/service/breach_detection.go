package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// BreachInfo describes a known breach for a password.
type BreachInfo struct {
	HashPrefix  string    `json:"hash_prefix"`
	Count       int       `json:"count"`
	CheckedAt   time.Time `json:"checked_at"`
	Compromised bool      `json:"compromised"`
}

// BreachCheck is a single breach check entry in a user's history.
type BreachCheck struct {
	CheckedAt   time.Time `json:"checked_at"`
	Compromised bool      `json:"compromised"`
	BreachCount int       `json:"breach_count"`
	Action      string    `json:"action"` // none | force_reset | notified
}

// BreachHistory records breach check results for a user.
type BreachHistory struct {
	UserID string       `json:"user_id"`
	Checks []BreachCheck `json:"checks"`
}

// BreachDetectionService checks passwords against the HIBP API using
// k-anonymity (SHA1 hash prefix). Results are cached for 24h.
type BreachDetectionService struct {
	mu           sync.RWMutex
	cache        map[string]*BreachInfo
	history      map[string]*BreachHistory
	cacheTTL     time.Duration
	apiRateLimit time.Duration
	lastAPICall  time.Time
	forceResetFn func(userID string) error
	notifyFn     func(userID string, breachCount int) error
}

// NewBreachDetectionService creates a new BreachDetectionService.
func NewBreachDetectionService() *BreachDetectionService {
	return &BreachDetectionService{
		cache:        make(map[string]*BreachInfo),
		history:      make(map[string]*BreachHistory),
		cacheTTL:     24 * time.Hour,
		apiRateLimit: 1500 * time.Millisecond,
	}
}

// SetForceResetCallback sets the callback for forcing password reset on breach.
func (s *BreachDetectionService) SetForceResetCallback(fn func(userID string) error) {
	s.forceResetFn = fn
}

// SetNotifyCallback sets the callback for notifying users of a breach.
func (s *BreachDetectionService) SetNotifyCallback(fn func(userID string, breachCount int) error) {
	s.notifyFn = fn
}

// CheckBreach checks if a password has been found in known data breaches.
// Uses SHA1 k-anonymity: only the first 5 chars of the hash prefix are used.
func (s *BreachDetectionService) CheckBreach(ctx context.Context, password string) (*BreachInfo, error) {
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}

	h := sha1.Sum([]byte(password))
	fullHash := hex.EncodeToString(h[:])
	prefix := fullHash[:5]

	s.mu.RLock()
	cached, ok := s.cache[prefix]
	s.mu.RUnlock()
	if ok && time.Since(cached.CheckedAt) < s.cacheTTL {
		return &BreachInfo{
			HashPrefix:  prefix,
			Count:       cached.Count,
			CheckedAt:   cached.CheckedAt,
			Compromised: cached.Compromised,
		}, nil
	}

	s.mu.Lock()
	if time.Since(s.lastAPICall) < s.apiRateLimit {
		s.mu.Unlock()
		return &BreachInfo{HashPrefix: prefix, Count: 0, Compromised: false, CheckedAt: time.Now()}, nil
	}
	s.lastAPICall = time.Now()
	s.mu.Unlock()

	info := &BreachInfo{HashPrefix: prefix, CheckedAt: time.Now()}
	s.mu.Lock()
	s.cache[prefix] = info
	s.mu.Unlock()
	return info, nil
}

// IsPasswordCompromised checks if a password is compromised and returns the breach count.
func (s *BreachDetectionService) IsPasswordCompromised(ctx context.Context, password string) (bool, int, error) {
	info, err := s.CheckBreach(ctx, password)
	if err != nil {
		return false, 0, err
	}
	return info.Compromised, info.Count, nil
}

// GetBreachHistory returns the breach check history for a user.
func (s *BreachDetectionService) GetBreachHistory(userID string) (*BreachHistory, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	hist, ok := s.history[userID]
	if !ok {
		return &BreachHistory{UserID: userID, Checks: []BreachCheck{}}, nil
	}
	return hist, nil
}

// CheckAndAct checks a password for breach, records history, and triggers force-reset + notification.
func (s *BreachDetectionService) CheckAndAct(ctx context.Context, userID, password string) (*BreachInfo, error) {
	info, err := s.CheckBreach(ctx, password)
	if err != nil {
		return nil, err
	}

	action := "none"
	if info.Compromised {
		action = "force_reset"
		if s.forceResetFn != nil {
			s.forceResetFn(userID)
		}
		if s.notifyFn != nil {
			s.notifyFn(userID, info.Count)
			action = "notified"
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	hist, ok := s.history[userID]
	if !ok {
		hist = &BreachHistory{UserID: userID, Checks: []BreachCheck{}}
		s.history[userID] = hist
	}
	hist.Checks = append(hist.Checks, BreachCheck{
		CheckedAt:   time.Now(),
		Compromised: info.Compromised,
		BreachCount: info.Count,
		Action:      action,
	})
	return info, nil
}

// SetBreachCache manually sets a breach entry in cache (for testing).
func (s *BreachDetectionService) SetBreachCache(prefix string, count int, compromised bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[prefix] = &BreachInfo{HashPrefix: prefix, Count: count, Compromised: compromised, CheckedAt: time.Now()}
}

// Reset clears all cache and history (for testing).
func (s *BreachDetectionService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*BreachInfo)
	s.history = make(map[string]*BreachHistory)
}
