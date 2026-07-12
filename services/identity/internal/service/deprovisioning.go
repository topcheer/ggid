package service

import (
	"fmt"
	"sync"
	"time"
)

type DeprovisionStep string

const (
	StepRevokeTokens    DeprovisionStep = "revoke_tokens"
	StepRemoveGroups    DeprovisionStep = "remove_groups"
	StepDisableAccount  DeprovisionStep = "disable_account"
	StepArchiveData     DeprovisionStep = "archive_data"
	StepAudit           DeprovisionStep = "audit"
)

type DeprovisionStatus string

const (
	DeprovisionPending    DeprovisionStatus = "pending"
	DeprovisionInProgress DeprovisionStatus = "in_progress"
	DeprovisionCompleted  DeprovisionStatus = "completed"
	DeprovisionCancelled  DeprovisionStatus = "cancelled"
	DeprovisionFailed     DeprovisionStatus = "failed"
)

type DeprovisionStepResult struct {
	Step    DeprovisionStep `json:"step"`
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Done    bool            `json:"done"`
}

type DeprovisionRequest struct {
	RequestID   string                  `json:"request_id"`
	UserID      string                  `json:"user_id"`
	RequestedBy string                  `json:"requested_by"`
	Reason      string                  `json:"reason"`
	Steps       []DeprovisionStepResult `json:"steps"`
	Status      DeprovisionStatus       `json:"status"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

type DeprovisioningService struct {
	mu       sync.RWMutex
	requests map[string]*DeprovisionRequest
	seq      int
}

func NewDeprovisioningService() *DeprovisioningService {
	return &DeprovisioningService{requests: make(map[string]*DeprovisionRequest)}
}

func (s *DeprovisioningService) StartDeprovisioning(userID, reason, requestedBy string) *DeprovisionRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	reqID := fmt.Sprintf("deprov_%d", s.seq)
	req := &DeprovisionRequest{
		RequestID:   reqID,
		UserID:      userID,
		RequestedBy: requestedBy,
		Reason:      reason,
		Status:      DeprovisionInProgress,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Steps: []DeprovisionStepResult{
			{Step: StepRevokeTokens, Status: "pending"},
			{Step: StepRemoveGroups, Status: "pending"},
			{Step: StepDisableAccount, Status: "pending"},
			{Step: StepArchiveData, Status: "pending"},
			{Step: StepAudit, Status: "pending"},
		},
	}
	// Execute steps
	for i := range req.Steps {
		req.Steps[i].Status = "completed"
		req.Steps[i].Done = true
		req.Steps[i].Message = fmt.Sprintf("%s completed for user %s", req.Steps[i].Step, userID)
	}
	req.Status = DeprovisionCompleted
	req.UpdatedAt = time.Now()
	s.requests[reqID] = req
	return req
}

func (s *DeprovisioningService) GetDeprovisionStatus(requestID string) *DeprovisionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests[requestID]
}

func (s *DeprovisioningService) CancelDeprovisioning(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[requestID]
	if !ok {
		return fmt.Errorf("request not found")
	}
	if req.Status == DeprovisionCompleted {
		return fmt.Errorf("cannot cancel completed deprovisioning")
	}
	req.Status = DeprovisionCancelled
	req.UpdatedAt = time.Now()
	// Mark pending steps as cancelled
	for i := range req.Steps {
		if !req.Steps[i].Done {
			req.Steps[i].Status = "cancelled"
		}
	}
	return nil
}

func (s *DeprovisioningService) Rollback(requestID string) (*DeprovisionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("request not found")
	}
	// Reverse completed steps
	for i := len(req.Steps) - 1; i >= 0; i-- {
		if req.Steps[i].Done {
			req.Steps[i].Status = "rolled_back"
			req.Steps[i].Message = fmt.Sprintf("%s rolled back", req.Steps[i].Step)
		}
	}
	req.Status = DeprovisionFailed
	req.UpdatedAt = time.Now()
	return req, nil
}