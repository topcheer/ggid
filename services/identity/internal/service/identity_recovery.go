package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type RecoveryMethod string

const (
	RecoveryEmail  RecoveryMethod = "email"
	RecoveryPhone  RecoveryMethod = "phone"
	RecoveryBackup RecoveryMethod = "backup_codes"
)

type RecoveryStatus string

const (
	RecoveryInitiated RecoveryStatus = "initiated"
	RecoveryVerified  RecoveryStatus = "verified"
	RecoveryCompleted RecoveryStatus = "completed"
	RecoveryExpired   RecoveryStatus = "expired"
	RecoveryCancelled RecoveryStatus = "cancelled"
)

type RecoveryRequest struct {
	RequestID   string         `json:"request_id"`
	UserID      string         `json:"user_id"`
	Method      RecoveryMethod `json:"method"`
	Token       string         `json:"token"`
	Status      RecoveryStatus `json:"status"`
	ExpiresAt   time.Time      `json:"expires_at"`
	WaitUntil   time.Time      `json:"wait_until"` // time-delayed recovery
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`
}

type RecoveryAuditEntry struct {
	RequestID string         `json:"request_id"`
	UserID    string         `json:"user_id"`
	Action    string         `json:"action"`
	Method    RecoveryMethod `json:"method"`
	Timestamp time.Time      `json:"timestamp"`
}

type IdentityRecoveryService struct {
	mu       sync.RWMutex
	requests map[string]*RecoveryRequest
	audit    []RecoveryAuditEntry
	seq      int
}

const maxRecoveryAuditEntries = 1000

// appendAudit appends an entry and caps the slice to prevent unbounded growth.
func (s *IdentityRecoveryService) appendAudit(entry RecoveryAuditEntry) {
	s.audit = append(s.audit, entry)
	if len(s.audit) > maxRecoveryAuditEntries {
		s.audit = s.audit[len(s.audit)-maxRecoveryAuditEntries:]
	}
}

func NewIdentityRecoveryService() *IdentityRecoveryService {
	return &IdentityRecoveryService{
		requests: make(map[string]*RecoveryRequest),
	}
}

func (s *IdentityRecoveryService) InitiateRecovery(userID string, method RecoveryMethod) (*RecoveryRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	reqID := fmt.Sprintf("rec_%d", s.seq)
	token, err := generateRecoveryToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery token: %w", err)
	}
	req := &RecoveryRequest{
		RequestID: reqID,
		UserID:    userID,
		Method:    method,
		Token:     token,
		Status:    RecoveryInitiated,
		ExpiresAt: time.Now().Add(48 * time.Hour), // must exceed WaitUntil
		WaitUntil: time.Now().Add(24 * time.Hour), // 24h time-delayed recovery
		CreatedAt: time.Now(),
	}
	s.requests[reqID] = req
	s.appendAudit(RecoveryAuditEntry{
		RequestID: reqID, UserID: userID, Action: "initiate", Method: method, Timestamp: time.Now(),
	})
	return req, nil
}

func (s *IdentityRecoveryService) VerifyRecoveryToken(userID, token string) (*RecoveryRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, req := range s.requests {
		if req.UserID == userID && subtle.ConstantTimeCompare([]byte(req.Token), []byte(token)) == 1 {
			if req.Status != RecoveryInitiated {
				return nil, fmt.Errorf("recovery request not in initiated state")
			}
			if time.Now().After(req.ExpiresAt) {
				return nil, fmt.Errorf("recovery token expired")
			}
			return req, nil
		}
	}
	return nil, fmt.Errorf("recovery token not found")
}

func (s *IdentityRecoveryService) CompleteRecovery(requestID string, newCredential string) (*RecoveryRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("recovery request not found")
	}
	if req.Status != RecoveryInitiated {
		return nil, fmt.Errorf("recovery request not in initiated state")
	}
	// Check time-delayed recovery wait period
	if time.Now().Before(req.WaitUntil) {
		return nil, fmt.Errorf("recovery wait period not elapsed, wait until %s", req.WaitUntil)
	}
	req.Status = RecoveryCompleted
	req.CompletedAt = time.Now()
	s.appendAudit(RecoveryAuditEntry{
		RequestID: requestID, UserID: req.UserID, Action: "complete", Method: req.Method, Timestamp: time.Now(),
	})
	return req, nil
}

func (s *IdentityRecoveryService) CancelRecovery(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}
	if req.Status == RecoveryCompleted {
		return fmt.Errorf("cannot cancel completed recovery")
	}
	req.Status = RecoveryCancelled
	s.appendAudit(RecoveryAuditEntry{
		RequestID: requestID, UserID: req.UserID, Action: "cancel", Method: req.Method, Timestamp: time.Now(),
	})
	return nil
}

func (s *IdentityRecoveryService) GetRecoveryAuditTrail() []RecoveryAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.audit
}

func (s *IdentityRecoveryService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	now := time.Now()
	for id, req := range s.requests {
		if now.After(req.ExpiresAt) && req.Status == RecoveryInitiated {
			req.Status = RecoveryExpired
			count++
		}
		// Remove expired or completed requests from memory to prevent unbounded growth.
		if now.After(req.ExpiresAt) {
			delete(s.requests, id)
		}
	}
	return count
}

// generateRecoveryToken creates a cryptographically random recovery token
// with at least 256 bits of entropy (32 random bytes, base64url-encoded).
func generateRecoveryToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rtok_" + base64.RawURLEncoding.EncodeToString(b), nil
}
