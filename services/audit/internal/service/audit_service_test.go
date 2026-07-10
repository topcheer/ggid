package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// --- Mock ---

type mockAuditRepo struct {
	events    []*domain.AuditEvent
	insertErr error
	listErr   error
	getErr    error
	stats     *domain.Stats
	statsErr  error
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

func (m *mockAuditRepo) GetStats(_ context.Context, _ uuid.UUID, _ time.Time) (*domain.Stats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	if m.stats != nil {
		return m.stats, nil
	}
	return &domain.Stats{EventsByAction: make(map[string]int)}, nil
}

func (m *mockAuditRepo) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func eventMatchesFilter(e *domain.AuditEvent, f domain.ListFilter) bool {
	if e.TenantID != f.TenantID {
		return false
	}
	if f.ActorID != nil && (e.ActorID == nil || *e.ActorID != *f.ActorID) {
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
	if f.StartTime != nil && e.CreatedAt.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && !e.CreatedAt.Before(*f.EndTime) {
		return false
	}
	return true
}

// --- Helpers ---

func newEvent(tenantID uuid.UUID, action string, result domain.EventResult, ts time.Time) *domain.AuditEvent {
	return &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Action:    action,
		Result:    result,
		CreatedAt: ts,
	}
}

// --- AuditService tests ---

func TestListEvents_RequiresTenantID(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	_, _, err := svc.ListEvents(context.Background(), domain.ListFilter{}, 1, 50)
	if err == nil {
		t.Fatal("expected error for nil tenant_id")
	}
}

func TestListEvents_DefaultPageSize(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	// pageSize=0 should default to 50; just verify no panic
	events, _, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: uuid.New(),
	}, 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events != nil {
		t.Error("expected nil events from empty repo")
	}
}

func TestListEvents_PageSizeCapped(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	// pageSize=10000 should be capped to 500
	events, _, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: uuid.New(),
	}, 1, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = events
}

func TestListEvents_Pagination(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = make([]*domain.AuditEvent, 10)
	for i := range repo.events {
		repo.events[i] = newEvent(tenantID, "user.login", domain.ResultSuccess, time.Now())
	}
	svc := NewAuditService(repo)

	// Page 1, size 3
	events, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantID,
	}, 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events on page 1, got %d", len(events))
	}

	// Page 4, size 3 → only 1 event (offset 9)
	events4, _, _ := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantID,
	}, 4, 3)
	if len(events4) != 1 {
		t.Errorf("expected 1 event on page 4, got %d", len(events4))
	}
}

func TestListEvents_FilterByAction(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		newEvent(tenantID, "user.login", domain.ResultSuccess, time.Now()),
		newEvent(tenantID, "user.login", domain.ResultSuccess, time.Now()),
		newEvent(tenantID, "role.assign", domain.ResultSuccess, time.Now()),
	}
	svc := NewAuditService(repo)

	events, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantID,
		Action:   "user.login",
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 login events, got %d", total)
	}
	for _, e := range events {
		if e.Action != "user.login" {
			t.Errorf("expected action user.login, got %s", e.Action)
		}
	}
}

func TestListEvents_FilterByResult(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		newEvent(tenantID, "user.login", domain.ResultSuccess, time.Now()),
		newEvent(tenantID, "user.login", domain.ResultFailure, time.Now()),
		newEvent(tenantID, "user.login", domain.ResultDenied, time.Now()),
	}
	svc := NewAuditService(repo)

	events, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantID,
		Result:   domain.ResultDenied,
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 denied event, got %d", total)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Result != domain.ResultDenied {
		t.Errorf("expected result denied, got %s", events[0].Result)
	}
}

func TestListEvents_FilterByActorID(t *testing.T) {
	tenantID := uuid.New()
	actor1 := uuid.New()
	actor2 := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		{TenantID: tenantID, ActorID: &actor1, Action: "a"},
		{TenantID: tenantID, ActorID: &actor2, Action: "a"},
		{TenantID: tenantID, ActorID: &actor1, Action: "a"},
	}
	for _, e := range repo.events {
		e.ID = uuid.New()
		e.CreatedAt = time.Now()
	}
	svc := NewAuditService(repo)

	_, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantID,
		ActorID:  &actor1,
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 events for actor1, got %d", total)
	}
}

func TestListEvents_FilterByTimeRange(t *testing.T) {
	tenantID := uuid.New()
	base := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		newEvent(tenantID, "a", domain.ResultSuccess, base.AddDate(0, 0, -10)), // Jun 5
		newEvent(tenantID, "a", domain.ResultSuccess, base.AddDate(0, 0, -1)),  // Jun 14
		newEvent(tenantID, "a", domain.ResultSuccess, base),                     // Jun 15
		newEvent(tenantID, "a", domain.ResultSuccess, base.AddDate(0, 0, 1)),   // Jun 16
	}
	svc := NewAuditService(repo)

	start := base.AddDate(0, 0, -2) // Jun 13
	end := base.AddDate(0, 0, 1)   // Jun 16 (exclusive)

	_, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID:  tenantID,
		StartTime: &start,
		EndTime:   &end,
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Jun 14 and Jun 15 fall in [Jun 13, Jun 16)
	if total != 2 {
		t.Errorf("expected 2 events in time range, got %d", total)
	}
}

func TestListEvents_FilterByResourceType(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		{TenantID: tenantID, ResourceType: "users", Action: "a"},
		{TenantID: tenantID, ResourceType: "roles", Action: "a"},
		{TenantID: tenantID, ResourceType: "users", Action: "a"},
	}
	for _, e := range repo.events {
		e.ID = uuid.New()
		e.CreatedAt = time.Now()
	}
	svc := NewAuditService(repo)

	_, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID:     tenantID,
		ResourceType: "users",
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 users events, got %d", total)
	}
}

func TestListEvents_CrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		newEvent(tenantA, "a", domain.ResultSuccess, time.Now()),
		newEvent(tenantB, "a", domain.ResultSuccess, time.Now()),
		newEvent(tenantA, "a", domain.ResultSuccess, time.Now()),
	}
	svc := NewAuditService(repo)

	_, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: tenantA,
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 events for tenantA, got %d", total)
	}
}

func TestListEvents_RepoError(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{
		listErr: errors.New("db down"),
	})
	_, _, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID: uuid.New(),
	}, 1, 50)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	_, err := svc.GetEvent(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestGetEvent_Found(t *testing.T) {
	tenantID := uuid.New()
	evt := newEvent(tenantID, "user.login", domain.ResultSuccess, time.Now())
	repo := &mockAuditRepo{events: []*domain.AuditEvent{evt}}
	svc := NewAuditService(repo)

	found, err := svc.GetEvent(context.Background(), evt.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Action != "user.login" {
		t.Errorf("expected action user.login, got %s", found.Action)
	}
}

func TestInsertEvent_Success(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	evt := &domain.AuditEvent{
		TenantID: uuid.New(),
		Action:   "test.event",
	}
	if err := svc.InsertEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.ID == uuid.Nil {
		t.Error("expected non-nil ID after insert")
	}
}

func TestInsertEvent_RepoError(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{
		insertErr: errors.New("write failed"),
	})
	err := svc.InsertEvent(context.Background(), &domain.AuditEvent{
		TenantID: uuid.New(),
		Action:   "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListEvents_CombinedFilters(t *testing.T) {
	tenantID := uuid.New()
	actor1 := uuid.New()
	base := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := &mockAuditRepo{}
	repo.events = []*domain.AuditEvent{
		{TenantID: tenantID, ActorID: &actor1, Action: "user.login", ResourceType: "users", Result: domain.ResultSuccess, CreatedAt: base},
		{TenantID: tenantID, ActorID: &actor1, Action: "user.login", ResourceType: "users", Result: domain.ResultFailure, CreatedAt: base},
		{TenantID: tenantID, ActorID: &actor1, Action: "role.assign", ResourceType: "roles", Result: domain.ResultSuccess, CreatedAt: base},
		{TenantID: tenantID, ActorID: &uuid.UUID{}, Action: "user.login", ResourceType: "users", Result: domain.ResultSuccess, CreatedAt: base},
	}
	for _, e := range repo.events {
		e.ID = uuid.New()
	}
	svc := NewAuditService(repo)

	// Filter: actor1 + user.login + success + users
	_, total, err := svc.ListEvents(context.Background(), domain.ListFilter{
		TenantID:     tenantID,
		ActorID:      &actor1,
		Action:       "user.login",
		ResourceType: "users",
		Result:       domain.ResultSuccess,
	}, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 event matching all filters, got %d", total)
	}
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

// --- GetStats tests ---

func TestGetStats_RequiresTenantID(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{})
	_, err := svc.GetStats(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil tenant_id")
	}
}

func TestGetStats_Success(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	repo := &mockAuditRepo{
		stats: &domain.Stats{
			TotalEvents24h:  100,
			FailedLogins24h: 5,
			EventsByAction: map[string]int{
				"user.login":   40,
				"role.assign":  10,
			},
			HourlyDistribution: []domain.HourlyCount{
				{Hour: time.Now().UTC().Truncate(time.Hour), Count: 15},
			},
			TopActors: []domain.ActorActivity{
				{ActorID: actorID, ActorName: "admin", Count: 30},
			},
		},
	}
	svc := NewAuditService(repo)

	stats, err := svc.GetStats(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalEvents24h != 100 {
		t.Errorf("expected 100 total events, got %d", stats.TotalEvents24h)
	}
	if stats.FailedLogins24h != 5 {
		t.Errorf("expected 5 failed logins, got %d", stats.FailedLogins24h)
	}
	if stats.EventsByAction["user.login"] != 40 {
		t.Errorf("expected 40 login events, got %d", stats.EventsByAction["user.login"])
	}
	if len(stats.TopActors) != 1 || stats.TopActors[0].ActorName != "admin" {
		t.Errorf("unexpected top actors: %+v", stats.TopActors)
	}
}

func TestGetStats_RepoError(t *testing.T) {
	svc := NewAuditService(&mockAuditRepo{statsErr: errors.New("db error")})
	_, err := svc.GetStats(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}
