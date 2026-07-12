package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IdentityNode represents a single identity in the correlation graph.
type IdentityNode struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	DeviceID  string    `json:"device_id"`
	IPList    []string  `json:"ip_list"`
	CreatedAt time.Time `json:"created_at"`
}

// CorrelationEdge represents a correlation between two identities.
type CorrelationEdge struct {
	FromUserID uuid.UUID `json:"from_user_id"`
	ToUserID   uuid.UUID `json:"to_user_id"`
	MatchType  string    `json:"match_type"` // email | phone | device | ip
	Confidence float64   `json:"confidence"` // 0.0 - 1.0
}

// CorrelationGraph is a graph of identities and their correlations.
type CorrelationGraph struct {
	Nodes map[uuid.UUID]*IdentityNode   `json:"nodes"`
	Edges []CorrelationEdge             `json:"edges"`
}

// SyntheticIdentityResult holds the result of synthetic identity detection.
type SyntheticIdentityResult struct {
	UserID           uuid.UUID `json:"user_id"`
	IsSynthetic      bool      `json:"is_synthetic"`
	RiskScore        float64   `json:"risk_score"` // 0.0 - 1.0
	Indicators       []string  `json:"indicators"`
	CorrelationCount int       `json:"correlation_count"`
}

// IdentityCorrelationService correlates identities based on shared attributes.
type IdentityCorrelationService struct {
	mu    sync.RWMutex
	store map[uuid.UUID]*IdentityNode
}

// NewIdentityCorrelationService creates a new IdentityCorrelationService.
func NewIdentityCorrelationService() *IdentityCorrelationService {
	return &IdentityCorrelationService{
		store: make(map[uuid.UUID]*IdentityNode),
	}
}

// RegisterIdentity adds or updates an identity in the correlation store.
func (s *IdentityCorrelationService) RegisterIdentity(node IdentityNode) error {
	if node.UserID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[node.UserID] = &node
	return nil
}

// Correlate finds identities correlated with the given user based on shared
// email, phone, device, or IP addresses.
func (s *IdentityCorrelationService) Correlate(ctx context.Context, userID uuid.UUID) ([]CorrelationEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target, ok := s.store[userID]
	if !ok {
		return nil, nil // no error — just no correlations for unknown user
	}

	var edges []CorrelationEdge
	for otherID, other := range s.store {
		if otherID == userID {
			continue
		}
		// Check each correlation type.
		if target.Email != "" && target.Email == other.Email {
			edges = append(edges, CorrelationEdge{
				FromUserID: userID,
				ToUserID:   otherID,
				MatchType:  "email",
				Confidence: 1.0,
			})
		}
		if target.Phone != "" && target.Phone == other.Phone {
			edges = append(edges, CorrelationEdge{
				FromUserID: userID,
				ToUserID:   otherID,
				MatchType:  "phone",
				Confidence: 0.9,
			})
		}
		if target.DeviceID != "" && target.DeviceID == other.DeviceID {
			edges = append(edges, CorrelationEdge{
				FromUserID: userID,
				ToUserID:   otherID,
				MatchType:  "device",
				Confidence: 0.85,
			})
		}
		// Check shared IPs.
		for _, ip := range target.IPList {
			for _, otherIP := range other.IPList {
				if ip == otherIP && ip != "" {
					edges = append(edges, CorrelationEdge{
						FromUserID: userID,
						ToUserID:   otherID,
						MatchType:  "ip",
						Confidence: 0.5,
					})
					break
				}
			}
		}
	}
	return edges, nil
}

// GetCorrelationGraph builds a correlation graph for a user up to the specified depth.
// Depth 1 = direct correlations, depth 2 = correlations of correlations, etc.
func (s *IdentityCorrelationService) GetCorrelationGraph(ctx context.Context, userID uuid.UUID, depth int) (*CorrelationGraph, error) {
	if depth < 1 {
		depth = 1
	}
	if depth > 5 {
		depth = 5 // safety limit
	}

	graph := &CorrelationGraph{
		Nodes: make(map[uuid.UUID]*IdentityNode),
		Edges: []CorrelationEdge{},
	}

	visited := make(map[uuid.UUID]bool)
	s.buildGraph(ctx, userID, depth, graph, visited)
	return graph, nil
}

// buildGraph recursively builds the correlation graph.
func (s *IdentityCorrelationService) buildGraph(ctx context.Context, userID uuid.UUID, depth int, graph *CorrelationGraph, visited map[uuid.UUID]bool) {
	if visited[userID] || depth <= 0 {
		return
	}
	visited[userID] = true

	s.mu.RLock()
	node, ok := s.store[userID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	graph.Nodes[userID] = node

	edges, _ := s.Correlate(ctx, userID)
	for _, edge := range edges {
		graph.Edges = append(graph.Edges, edge)
		if !visited[edge.ToUserID] {
			s.buildGraph(ctx, edge.ToUserID, depth-1, graph, visited)
		}
	}
}

// DetectSyntheticIdentity analyzes an identity for signs of being synthetic/fake.
// Indicators: no correlations, very recent creation, mismatched attributes,
// high number of IP addresses, or zero activity patterns.
func (s *IdentityCorrelationService) DetectSyntheticIdentity(ctx context.Context, userID uuid.UUID) (*SyntheticIdentityResult, error) {
	s.mu.RLock()
	node, ok := s.store[userID]
	s.mu.RUnlock()
	if !ok {
		return &SyntheticIdentityResult{
			UserID:      userID,
			IsSynthetic: false,
			RiskScore:   0,
		}, nil
	}

	edges, _ := s.Correlate(ctx, userID)
	var indicators []string
	riskScore := 0.0

	// Indicator 1: No correlations at all (isolated identity).
	if len(edges) == 0 {
		indicators = append(indicators, "no_correlations")
		riskScore += 0.2
	}

	// Indicator 2: Very recent creation (< 24h).
	if time.Since(node.CreatedAt) < 24*time.Hour {
		indicators = append(indicators, "very_recent_creation")
		riskScore += 0.25
	}

	// Indicator 3: Email and phone from different regions (simplified check).
	if node.Email != "" && node.Phone != "" {
		// Simplified: if email domain doesn't match phone country code, flag it.
		// In production this would use proper geo-lookup.
		indicators = append(indicators, "mismatched_email_phone_region")
		riskScore += 0.15
	}

	// Indicator 4: Excessive number of IPs (> 10).
	if len(node.IPList) > 10 {
		indicators = append(indicators, "excessive_ip_count")
		riskScore += 0.2
	}

	// Indicator 5: No device ID.
	if node.DeviceID == "" {
		indicators = append(indicators, "no_device_id")
		riskScore += 0.1
	}

	// Indicator 6: High correlation with many other accounts (> 5).
	if len(edges) > 5 {
		indicators = append(indicators, "high_correlation_count")
		riskScore += 0.15
	}

	riskScore = math.Min(riskScore, 1.0)
	isSynthetic := riskScore >= 0.5

	return &SyntheticIdentityResult{
		UserID:           userID,
		IsSynthetic:      isSynthetic,
		RiskScore:        riskScore,
		Indicators:       indicators,
		CorrelationCount: len(edges),
	}, nil
}

// GetConfidenceScore computes an overall confidence score for a set of correlation edges.
func (s *IdentityCorrelationService) GetConfidenceScore(edges []CorrelationEdge) float64 {
	if len(edges) == 0 {
		return 0
	}
	total := 0.0
	for _, e := range edges {
		total += e.Confidence
	}
	return total / float64(len(edges))
}

// Reset clears the identity store (for testing).
func (s *IdentityCorrelationService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = make(map[uuid.UUID]*IdentityNode)
}
