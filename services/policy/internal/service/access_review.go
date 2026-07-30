package service

import (
	"fmt"
	"sync"
	"time"
)

type ReviewDecision string

const (
	ReviewApprove ReviewDecision = "approve"
	ReviewReject  ReviewDecision = "reject"
	ReviewPending ReviewDecision = "pending"
)

type AccessReviewRecord struct {
	ReviewID  string         `json:"review_id"`
	TenantID  string         `json:"tenant_id"`
	UserID    string         `json:"user_id"`
	Reviewer  string         `json:"reviewer"`
	Scopes    []string       `json:"scopes"`
	Roles     []string       `json:"roles"`
	Decision  ReviewDecision `json:"decision"`
	Comment   string         `json:"comment"`
	Timestamp time.Time      `json:"timestamp"`
	Scheduled bool           `json:"scheduled"`
}

type ReviewFilter struct {
	TenantID string `json:"tenant_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Reviewer string `json:"reviewer,omitempty"`
	Decision string `json:"decision,omitempty"`
}

type AccessReviewService struct {
	mu           sync.RWMutex
	reviews      map[string]*AccessReviewRecord
	byUser       map[string][]string
	seq          int
	intervalDays int
}

func NewAccessReviewService(intervalDays int) *AccessReviewService {
	return &AccessReviewService{
		reviews:      make(map[string]*AccessReviewRecord),
		byUser:       make(map[string][]string),
		intervalDays: intervalDays,
	}
}

func (s *AccessReviewService) CreateAccessReview(tenantID, userID, reviewer string, scopes, roles []string) *AccessReviewRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	r := &AccessReviewRecord{
		ReviewID:  fmt.Sprintf("rev_%d", s.seq),
		TenantID:  tenantID,
		UserID:    userID,
		Reviewer:  reviewer,
		Scopes:    scopes,
		Roles:     roles,
		Decision:  ReviewPending,
		Timestamp: time.Now(),
	}
	s.reviews[r.ReviewID] = r
	s.byUser[userID] = append(s.byUser[userID], r.ReviewID)
	return r
}

func (s *AccessReviewService) ListAccessReviews(filter ReviewFilter) []*AccessReviewRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*AccessReviewRecord
	for _, r := range s.reviews {
		if filter.TenantID != "" && r.TenantID != filter.TenantID {
			continue
		}
		if filter.UserID != "" && r.UserID != filter.UserID {
			continue
		}
		if filter.Reviewer != "" && r.Reviewer != filter.Reviewer {
			continue
		}
		if filter.Decision != "" && string(r.Decision) != filter.Decision {
			continue
		}
		list = append(list, r)
	}
	return list
}

func (s *AccessReviewService) GetAccessReview(id string) *AccessReviewRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reviews[id]
}

func (s *AccessReviewService) UpdateAccessReview(id string, decision ReviewDecision, comment string) *AccessReviewRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.reviews[id]
	if !ok {
		return nil
	}
	r.Decision = decision
	r.Comment = comment
	r.Timestamp = time.Now()
	return r
}

func (s *AccessReviewService) GenerateScheduledReviews(tenantID string, userIDs []string, reviewer string) []*AccessReviewRecord {
	var created []*AccessReviewRecord
	for _, uid := range userIDs {
		r := s.CreateAccessReview(tenantID, uid, reviewer, nil, nil)
		r.Scheduled = true
		created = append(created, r)
	}
	return created
}

func (s *AccessReviewService) GetReviewInterval() int {
	return s.intervalDays
}
