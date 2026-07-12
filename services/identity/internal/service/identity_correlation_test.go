package service

	import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIdentityCorrelation_CorrelateByEmail(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, Email: "shared@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, Email: "shared@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})

	edges, err := svc.Correlate(context.Background(), user1)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if len(edges) == 0 {
		t.Fatal("should find correlation by email")
	}
	if edges[0].MatchType != "email" {
		t.Errorf("expected match type 'email', got '%s'", edges[0].MatchType)
	}
	if edges[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", edges[0].Confidence)
	}
}

func TestIdentityCorrelation_CorrelateByPhone(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, Phone: "+1234567890", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, Phone: "+1234567890", CreatedAt: time.Now().Add(-48 * time.Hour)})

	edges, _ := svc.Correlate(context.Background(), user1)
	if len(edges) == 0 {
		t.Fatal("should find correlation by phone")
	}
	if edges[0].MatchType != "phone" {
		t.Errorf("expected match type 'phone', got '%s'", edges[0].MatchType)
	}
}

func TestIdentityCorrelation_CorrelateByDevice(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, DeviceID: "device-abc", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, DeviceID: "device-abc", CreatedAt: time.Now().Add(-48 * time.Hour)})

	edges, _ := svc.Correlate(context.Background(), user1)
	if len(edges) == 0 {
		t.Fatal("should find correlation by device")
	}
	if edges[0].MatchType != "device" {
		t.Errorf("expected match type 'device', got '%s'", edges[0].MatchType)
	}
}

func TestIdentityCorrelation_CorrelateByIP(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, IPList: []string{"1.2.3.4"}, CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, IPList: []string{"1.2.3.4"}, CreatedAt: time.Now().Add(-48 * time.Hour)})

	edges, _ := svc.Correlate(context.Background(), user1)
	if len(edges) == 0 {
		t.Fatal("should find correlation by IP")
	}
	if edges[0].MatchType != "ip" {
		t.Errorf("expected match type 'ip', got '%s'", edges[0].MatchType)
	}
}

func TestIdentityCorrelation_NoCorrelation(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, Email: "a@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, Email: "b@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})

	edges, _ := svc.Correlate(context.Background(), user1)
	if len(edges) != 0 {
		t.Errorf("expected 0 correlations, got %d", len(edges))
	}
}

func TestIdentityCorrelation_UnknownUser(t *testing.T) {
	svc := NewIdentityCorrelationService()
	edges, err := svc.Correlate(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if len(edges) != 0 {
		t.Error("unknown user should have 0 correlations")
	}
}

func TestIdentityCorrelation_GetCorrelationGraph(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	svc.RegisterIdentity(IdentityNode{UserID: user1, Email: "shared@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, Email: "shared@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user3, Phone: "+1234567890", CreatedAt: time.Now().Add(-48 * time.Hour)})
	svc.RegisterIdentity(IdentityNode{UserID: user2, Phone: "+1234567890", CreatedAt: time.Now().Add(-48 * time.Hour)})

	graph, err := svc.GetCorrelationGraph(context.Background(), user1, 2)
	if err != nil {
		t.Fatalf("GetCorrelationGraph: %v", err)
	}
	if len(graph.Nodes) == 0 {
		t.Error("graph should have nodes")
	}
	if _, ok := graph.Nodes[user1]; !ok {
		t.Error("user1 should be in graph")
	}
}

func TestIdentityCorrelation_DetectSyntheticIdentity_HighRisk(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user := uuid.New()

	// No correlations + recent creation + no device + many IPs = high risk.
	svc.RegisterIdentity(IdentityNode{
		UserID:    user,
		Email:     "new@example.com",
		Phone:     "+1234567890",
		DeviceID:  "",
		IPList:    []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4", "5.5.5.5", "6.6.6.6", "7.7.7.7", "8.8.8.8", "9.9.9.9", "10.10.10.10", "11.11.11.11"},
		CreatedAt: time.Now().Add(-1 * time.Hour), // very recent
	})

	result, err := svc.DetectSyntheticIdentity(context.Background(), user)
	if err != nil {
		t.Fatalf("DetectSyntheticIdentity: %v", err)
	}
	if !result.IsSynthetic {
		t.Error("should detect synthetic identity")
	}
	if result.RiskScore < 0.5 {
		t.Errorf("risk score should be >= 0.5, got %f", result.RiskScore)
	}
	if len(result.Indicators) == 0 {
		t.Error("should have risk indicators")
	}
}

func TestIdentityCorrelation_DetectSyntheticIdentity_LowRisk(t *testing.T) {
	svc := NewIdentityCorrelationService()
	user1 := uuid.New()
	user2 := uuid.New()

	// Established identity with a correlation.
	svc.RegisterIdentity(IdentityNode{
		UserID:    user1,
		Email:     "established@example.com",
		DeviceID:  "device-xyz",
		IPList:    []string{"1.1.1.1"},
		CreatedAt: time.Now().Add(-720 * time.Hour), // 30 days ago
	})
	svc.RegisterIdentity(IdentityNode{
		UserID:    user2,
		Email:     "established@example.com",
		DeviceID:  "device-xyz",
		IPList:    []string{"1.1.1.1"},
		CreatedAt: time.Now().Add(-720 * time.Hour),
	})

	result, err := svc.DetectSyntheticIdentity(context.Background(), user1)
	if err != nil {
		t.Fatalf("DetectSyntheticIdentity: %v", err)
	}
	if result.IsSynthetic {
		t.Error("established identity should not be synthetic")
	}
	if result.RiskScore >= 0.5 {
		t.Errorf("risk score should be < 0.5, got %f", result.RiskScore)
	}
}

func TestIdentityCorrelation_DetectSynthetic_UnknownUser(t *testing.T) {
	svc := NewIdentityCorrelationService()
	result, err := svc.DetectSyntheticIdentity(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("DetectSyntheticIdentity: %v", err)
	}
	if result.IsSynthetic {
		t.Error("unknown user should not be synthetic")
	}
}

func TestIdentityCorrelation_ConfidenceScore(t *testing.T) {
	svc := NewIdentityCorrelationService()
	edges := []CorrelationEdge{
		{Confidence: 1.0},
		{Confidence: 0.5},
		{Confidence: 0.8},
	}
	score := svc.GetConfidenceScore(edges)
	expected := (1.0 + 0.5 + 0.8) / 3.0
	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("expected %f, got %f", expected, score)
	}
}

func TestIdentityCorrelation_ConfidenceScore_Empty(t *testing.T) {
	svc := NewIdentityCorrelationService()
	score := svc.GetConfidenceScore(nil)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}
}

func TestIdentityCorrelation_RegisterNilUser(t *testing.T) {
	svc := NewIdentityCorrelationService()
	err := svc.RegisterIdentity(IdentityNode{UserID: uuid.Nil})
	if err == nil {
		t.Error("should error on nil user ID")
	}
}
