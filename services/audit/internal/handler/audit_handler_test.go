package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/ggid/ggid/api/gen/audit/v1"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Mock repo ---

type mockAuditRepo struct {
	events   []*domain.AuditEvent
	insertErr error
	listErr   error
	getErr    error
}

func (m *mockAuditRepo) Insert(_ context.Context, e *domain.AuditEvent) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	e.ID = uuid.New()
	m.events = append(m.events, e)
	return nil
}

func (m *mockAuditRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, e := range m.events {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAuditRepo) List(_ context.Context, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var filtered []*domain.AuditEvent
	for _, e := range m.events {
		if !eventMatchesFilter(e, filter) {
			continue
		}
		filtered = append(filtered, e)
	}
	total := len(filtered)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func eventMatchesFilter(e *domain.AuditEvent, f domain.ListFilter) bool {
	if e.TenantID != f.TenantID {
		return false
	}
	if f.Action != "" && e.Action != f.Action {
		return false
	}
	if f.ResourceType != "" && e.ResourceType != f.ResourceType {
		return false
	}
	if f.Result != "" && e.Result != f.Result {
		return false
	}
	return true
}

func strPtr(s string) *string { return &s }

// --- Tests ---

func TestListEvents_InvalidTenantID(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{})
	h := NewAuditHandler(svc)

	_, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error for invalid tenant_id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestListEvents_EmptyResult(t *testing.T) {
	tenantID := uuid.New()
	svc := service.NewAuditService(&mockAuditRepo{})
	h := NewAuditHandler(svc)

	resp, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 total, got %d", resp.Total)
	}
}

func TestListEvents_WithEvents(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{
		events: []*domain.AuditEvent{
			{ID: uuid.New(), TenantID: tenantID, Action: "user.login", Result: domain.ResultSuccess, CreatedAt: time.Now()},
			{ID: uuid.New(), TenantID: tenantID, Action: "role.assign", Result: domain.ResultSuccess, CreatedAt: time.Now()},
		},
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	resp, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 total, got %d", resp.Total)
	}
	if len(resp.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(resp.Events))
	}
}

func TestListEvents_FilterByAction(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{
		events: []*domain.AuditEvent{
			{ID: uuid.New(), TenantID: tenantID, Action: "user.login", Result: domain.ResultSuccess, CreatedAt: time.Now()},
			{ID: uuid.New(), TenantID: tenantID, Action: "role.assign", Result: domain.ResultSuccess, CreatedAt: time.Now()},
		},
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	resp, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: tenantID.String(),
		Action:   strPtr("user.login"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 total for login, got %d", resp.Total)
	}
}

func TestListEvents_FilterByResult(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{
		events: []*domain.AuditEvent{
			{ID: uuid.New(), TenantID: tenantID, Action: "user.login", Result: domain.ResultSuccess, CreatedAt: time.Now()},
			{ID: uuid.New(), TenantID: tenantID, Action: "user.login", Result: domain.ResultFailure, CreatedAt: time.Now()},
		},
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	resp, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: tenantID.String(),
		Result:   strPtr("failure"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 failure, got %d", resp.Total)
	}
	if resp.Events[0].Result != "failure" {
		t.Errorf("expected result failure, got %s", resp.Events[0].Result)
	}
}

func TestListEvents_InvalidActorID(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{})
	h := NewAuditHandler(svc)

	_, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: uuid.New().String(),
		ActorId:  strPtr("not-a-uuid"),
	})
	if err == nil {
		t.Fatal("expected error for invalid actor_id")
	}
}

func TestListEvents_Pagination(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{}
	for i := 0; i < 10; i++ {
		repo.events = append(repo.events, &domain.AuditEvent{
			ID: uuid.New(), TenantID: tenantID, Action: "test", Result: domain.ResultSuccess, CreatedAt: time.Now(),
		})
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	resp, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: tenantID.String(),
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 10 {
		t.Errorf("expected 10 total, got %d", resp.Total)
	}
}

func TestGetEvent_InvalidID(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{})
	h := NewAuditHandler(svc)

	_, err := h.GetEvent(context.Background(), &pb.GetEventRequest{
		Id: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	svc := service.NewAuditService(&mockAuditRepo{})
	h := NewAuditHandler(svc)

	_, err := h.GetEvent(context.Background(), &pb.GetEventRequest{
		Id: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected error for non-existent event")
	}
}

func TestGetEvent_Found(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()
	repo := &mockAuditRepo{
		events: []*domain.AuditEvent{
			{ID: eventID, TenantID: tenantID, Action: "user.login", ActorType: domain.ActorUser,
				ActorName: "test@example.com", Result: domain.ResultSuccess, CreatedAt: time.Now()},
		},
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	event, err := h.GetEvent(context.Background(), &pb.GetEventRequest{
		Id: eventID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Action != "user.login" {
		t.Errorf("expected action user.login, got %s", event.Action)
	}
	if event.ActorName != "test@example.com" {
		t.Errorf("expected actor_name test@example.com, got %s", event.ActorName)
	}
}

func TestGetEvent_WithMetadata(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()
	repo := &mockAuditRepo{
		events: []*domain.AuditEvent{
			{ID: eventID, TenantID: tenantID, Action: "test",
				Result:    domain.ResultSuccess,
				Metadata:  map[string]any{"key": "value", "count": float64(42)},
				CreatedAt: time.Now()},
		},
	}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	event, err := h.GetEvent(context.Background(), &pb.GetEventRequest{
		Id: eventID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	v := event.Metadata.Fields["key"].GetStringValue()
	if v != "value" {
		t.Errorf("expected metadata.key=value, got %s", v)
	}
}

func TestListEvents_RepoError(t *testing.T) {
	repo := &mockAuditRepo{listErr: errors.New("db down")}
	svc := service.NewAuditService(repo)
	h := NewAuditHandler(svc)

	_, err := h.ListEvents(context.Background(), &pb.ListEventsRequest{
		TenantId: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestEventToProto_AllFields(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	resourceID := uuid.New()
	now := time.Now()

	e := &domain.AuditEvent{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ActorType:    domain.ActorSystem,
		ActorID:      &actorID,
		ActorName:    "system",
		Action:       "user.create",
		ResourceType: "users",
		ResourceID:   &resourceID,
		ResourceName: "john.doe",
		Result:       domain.ResultSuccess,
		IPAddress:    "10.0.0.1",
		UserAgent:    "test-agent/1.0",
		RequestID:    "req-123",
		Metadata:     map[string]any{"source": "api"},
		CreatedAt:    now,
	}

	p := eventToProto(e)

	if p.Id != e.ID.String() {
		t.Errorf("id mismatch")
	}
	if p.TenantId != tenantID.String() {
		t.Errorf("tenant_id mismatch")
	}
	if p.ActorType != "system" {
		t.Errorf("actor_type mismatch")
	}
	if p.ActorId != actorID.String() {
		t.Errorf("actor_id mismatch")
	}
	if p.ResourceId != resourceID.String() {
		t.Errorf("resource_id mismatch")
	}
	if p.IpAddress != "10.0.0.1" {
		t.Errorf("ip_address mismatch")
	}
	if p.Metadata == nil {
		t.Error("expected non-nil metadata")
	}
}

func TestEventToProto_NilIDs(t *testing.T) {
	e := &domain.AuditEvent{
		Action: "test",
		Result: domain.ResultSuccess,
	}

	p := eventToProto(e)
	if p.ActorId != "" {
		t.Errorf("expected empty actor_id for Nil UUID, got %s", p.ActorId)
	}
	if p.ResourceId != "" {
		t.Errorf("expected empty resource_id for Nil UUID, got %s", p.ResourceId)
	}
}
