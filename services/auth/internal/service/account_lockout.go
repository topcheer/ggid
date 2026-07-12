package service

import (
	"sync"
	"time"
)

type LockoutConfig struct {
	MaxAttempts      int           `json:"max_attempts"`
	WindowMinutes    int           `json:"window_minutes"`
	LockoutDuration  time.Duration `json:"lockout_duration"`
	CaptchaThreshold int           `json:"captcha_threshold"`
}

type LockoutState struct {
	UserID      string    `json:"user_id"`
	Locked      bool      `json:"locked"`
	UnlockAt    time.Time `json:"unlock_at"`
	Attempts    int       `json:"attempts"`
	Reason      string    `json:"reason"`
	LockedBy    string    `json:"locked_by"`
}

type LockoutDecision string

const (
	LockoutUnlocked        LockoutDecision = "unlocked"
	LockoutLocked          LockoutDecision = "locked"
	LockoutCaptchaRequired LockoutDecision = "captcha_required"
)

type LoginLockoutService struct {
	mu       sync.RWMutex
	config   LockoutConfig
	users    map[string]*LockoutState
	ipAttempts map[string]int
}

func NewLoginLockoutService(config LockoutConfig) *LoginLockoutService {
	return &LoginLockoutService{
		config:     config,
		users:      make(map[string]*LockoutState),
		ipAttempts: make(map[string]int),
	}
}

func (s *LoginLockoutService) RecordFailedAttempt(userID, ip string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.users[userID]
	if !ok {
		state = &LockoutState{UserID: userID}
		s.users[userID] = state
	}
	state.Attempts++
	s.ipAttempts[ip]++

	if s.config.MaxAttempts > 0 && state.Attempts >= s.config.MaxAttempts {
		state.Locked = true
		state.UnlockAt = time.Now().Add(s.config.LockoutDuration)
		state.Reason = "max attempts exceeded"
	}
	return state.Attempts
}

func (s *LoginLockoutService) CheckLockout(userID string) (LockoutDecision, *LockoutState) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.users[userID]
	if !ok {
		return LockoutUnlocked, nil
	}
	if state.Locked {
		if time.Now().After(state.UnlockAt) {
			return LockoutUnlocked, state
		}
		return LockoutLocked, state
	}
	if s.config.CaptchaThreshold > 0 && state.Attempts >= s.config.CaptchaThreshold {
		return LockoutCaptchaRequired, state
	}
	return LockoutUnlocked, state
}

func (s *LoginLockoutService) LockUser(userID string, duration time.Duration, reason, lockedBy string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.users[userID]
	if !ok {
		state = &LockoutState{UserID: userID}
		s.users[userID] = state
	}
	state.Locked = true
	state.UnlockAt = time.Now().Add(duration)
	state.Reason = reason
	state.LockedBy = lockedBy
}

func (s *LoginLockoutService) UnlockUser(userID, unlockerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.users[userID]; ok {
		state.Locked = false
		state.Attempts = 0
		state.UnlockAt = time.Time{}
		state.Reason = ""
		state.LockedBy = ""
	}
}

func (s *LoginLockoutService) ResetAttempts(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.users[userID]; ok {
		state.Attempts = 0
	}
}

func (s *LoginLockoutService) AutoUnlockExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	now := time.Now()
	for _, state := range s.users {
		if state.Locked && now.After(state.UnlockAt) {
			state.Locked = false
			state.Attempts = 0
			count++
		}
	}
	return count
}