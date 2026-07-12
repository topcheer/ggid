package service

import (
	"sync"
	"time"
)

type LoginSecurityConfig struct {
	MaxAttempts          int           `json:"max_attempts"`
	LockoutDuration      time.Duration `json:"lockout_duration"`
	CaptchaAfterAttempts int           `json:"captcha_after_attempts"`
	IPAllowlist          []string      `json:"ip_allowlist"`
	IPBlocklist          []string      `json:"ip_blocklist"`
	EnforceMFAForAdmin   bool          `json:"enforce_mfa_for_admin"`
}

type LoginDecision string

const (
	LoginAllow   LoginDecision = "allow"
	LoginDeny    LoginDecision = "deny"
	LoginCaptcha LoginDecision = "captcha"
	LoginStepUp  LoginDecision = "step_up"
)

type LoginSecurityService struct {
	mu       sync.RWMutex
	config   LoginSecurityConfig
	locked   map[string]time.Time // userID -> unlock time
	attempts map[string]int       // userID -> attempt count
}

func NewLoginSecurityService(config LoginSecurityConfig) *LoginSecurityService {
	return &LoginSecurityService{
		config:   config,
		locked:   make(map[string]time.Time),
		attempts: make(map[string]int),
	}
}

func (s *LoginSecurityService) CheckLoginPolicy(userID, ip string, attempts int, isAdmin bool) (LoginDecision, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check IP blocklist
	for _, blocked := range s.config.IPBlocklist {
		if ip == blocked {
			return LoginDeny, "IP blocked"
		}
	}

	// Check IP allowlist (if set, only allowlisted IPs allowed)
	if len(s.config.IPAllowlist) > 0 {
		allowed := false
		for _, a := range s.config.IPAllowlist {
			if ip == a {
				allowed = true
				break
			}
		}
		if !allowed {
			return LoginDeny, "IP not in allowlist"
		}
	}

	// Check account lock
	if unlockAt, locked := s.locked[userID]; locked {
		if time.Now().Before(unlockAt) {
			return LoginDeny, "account locked"
		}
		delete(s.locked, userID)
	}

	// Check attempts
	s.attempts[userID] = attempts

	// Captcha threshold
	if s.config.CaptchaAfterAttempts > 0 && attempts >= s.config.CaptchaAfterAttempts {
		return LoginCaptcha, "captcha required"
	}

	// Max attempts → lock
	if s.config.MaxAttempts > 0 && attempts >= s.config.MaxAttempts {
		s.locked[userID] = time.Now().Add(s.config.LockoutDuration)
		return LoginDeny, "account locked due to max attempts"
	}

	// MFA for admin
	if s.config.EnforceMFAForAdmin && isAdmin {
		return LoginStepUp, "MFA required for admin"
	}

	// Anomaly detection: rapid increase in attempts
	if attempts >= 5 && attempts < s.config.MaxAttempts {
		return LoginCaptcha, "anomalous login pattern detected"
	}

	return LoginAllow, ""
}

func (s *LoginSecurityService) LockAccount(userID string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locked[userID] = time.Now().Add(duration)
}

func (s *LoginSecurityService) UnlockAccount(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.locked, userID)
	delete(s.attempts, userID)
}

func (s *LoginSecurityService) IsAccountLocked(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	unlockAt, locked := s.locked[userID]
	return locked && time.Now().Before(unlockAt)
}