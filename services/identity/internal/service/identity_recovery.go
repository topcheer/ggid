package service

import (
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
	RequestID  string         `json:"request_id"`
	UserID     string         `json:"user_id"`
	Method     RecoveryMethod `json:"method"`
	Token      string         `json:"token"`
	Status     RecoveryStatus `json:"status"`
	ExpiresAt  time.Time      `json:"expires_at"`
	WaitUntil  time.Time      `json:"wait_until"` // time-delayed recovery
	CreatedAt  time.Time      `json:"created_at"`
	CompletedAt time.Time     `json:"completed_at,omitempty"`
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
	token := fmt.Sprintf("rtok_%d_%d", s.seq, time.Now().UnixNano())
	req := &RecoveryRequest{
		RequestID: reqID,
		UserID:    userID,
		Method:    method,
		Token:     token,
		Status:    RecoveryInitiated,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		WaitUntil: time.Now().Add(24 * time.Hour), // 24h time-delayed recovery
		CreatedAt: time.Now(),
	}
	s.requests[reqID] = req
	s.audit = append(s.audit, RecoveryAuditEntry{
		RequestID: reqID, UserID: userID, Action: "initiate", Method: method, Timestamp: time.Now(),
	})
	return req, nil
}

func (s *IdentityRecoveryService) VerifyRecoveryToken(userID, token string) (*RecoveryRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, req := range s.requests {
		if req.UserID == userID && req.Token == token {
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
	s.audit = append(s.audit, RecoveryAuditEntry{
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
	s.audit = append(s.audit, RecoveryAuditEntry{
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
	for _, req := range s.requests {
		if now.After(req.ExpiresAt) && req.Status == RecoveryInitiated {
			req.Status = RecoveryExpired
			count++
		}
	}
	return count
}