package retention

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockDeleter implements EventDeleter for testing.
type mockDeleter struct {
	events       []time.Time // timestamps of events
	deleteOldErr error
	countErr     error
	deleteExErr  error
}

func (m *mockDeleter) DeleteOlderThan(_ context.Context, before time.Time) (int64, error) {
	if m.deleteOldErr != nil {
		return 0, m.deleteOldErr
	}
	var deleted int64
	for _, ts := range m.events {
		if ts.Before(before) {
			deleted++
		}
	}
	return deleted, nil
}

func (m *mockDeleter) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.events)), nil
}

func (m *mockDeleter) DeleteExcess(_ context.Context, keep int64) (int64, error) {
	if m.deleteExErr != nil {
		return 0, m.deleteExErr
	}
	if int64(len(m.events)) <= keep {
		return 0, nil
	}
	return int64(len(m.events)) - keep, nil
}

// 1. TestApply_Disabled
func TestApply_Disabled(t *testing.T) {
	p := &RetentionPolicy{Enabled: false}
	d := &mockDeleter{events: []time.Time{time.Now()}}
	r, err := p.Apply(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.TotalDeleted != 0 {
		t.Fatalf("expected 0 deleted for disabled policy, got %d", r.TotalDeleted)
	}
}

// 2. TestApply_AgeBased
func TestApply_AgeBased(t *testing.T) {
	now := time.Now()
	p := &RetentionPolicy{
		MaxAge:  30 * 24 * time.Hour,
		Enabled: true,
	}
	d := &mockDeleter{
		events: []time.Time{
			now.Add(-10 * 24 * time.Hour),  // 10 days old — keep
			now.Add(-50 * 24 * time.Hour),  // 50 days old — delete
			now.Add(-100 * 24 * time.Hour), // 100 days old — delete
		},
	}
	r, err := p.Apply(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.DeletedByAge != 2 {
		t.Fatalf("expected 2 deleted by age, got %d", r.DeletedByAge)
	}
}

// 3. TestApply_CountBased
func TestApply_CountBased(t *testing.T) {
	now := time.Now()
	p := &RetentionPolicy{
		MaxEvents: 2,
		Enabled:   true,
	}
	d := &mockDeleter{
		events: []time.Time{now, now, now, now, now}, // 5 events
	}
	r, err := p.Apply(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.DeletedByCount != 3 {
		t.Fatalf("expected 3 deleted by count, got %d", r.DeletedByCount)
	}
}

// 4. TestApply_BothAgeAndCount
func TestApply_BothAgeAndCount(t *testing.T) {
	now := time.Now()
	p := &RetentionPolicy{
		MaxAge:    30 * 24 * time.Hour,
		MaxEvents: 2,
		Enabled:   true,
	}
	d := &mockDeleter{
		events: []time.Time{
			now,
			now.Add(-50 * 24 * time.Hour),
			now.Add(-100 * 24 * time.Hour),
		},
	}
	r, err := p.Apply(context.Background(), d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.DeletedByAge != 2 {
		t.Fatalf("expected 2 by age, got %d", r.DeletedByAge)
	}
	if r.TotalDeleted < 2 {
		t.Fatalf("expected total >=2, got %d", r.TotalDeleted)
	}
}

// 5. TestApply_DeleteOlderError
func TestApply_DeleteOlderError(t *testing.T) {
	p := &RetentionPolicy{MaxAge: time.Hour, Enabled: true}
	d := &mockDeleter{deleteOldErr: errors.New("db error")}
	_, err := p.Apply(context.Background(), d)
	if err == nil {
		t.Fatal("expected error")
	}
}

// 6. TestApply_CountError
func TestApply_CountError(t *testing.T) {
	p := &RetentionPolicy{MaxEvents: 10, Enabled: true}
	d := &mockDeleter{countErr: errors.New("count failed")}
	_, err := p.Apply(context.Background(), d)
	if err == nil {
		t.Fatal("expected error")
	}
}

// 7. TestNewDefaultPolicy
func TestNewDefaultPolicy(t *testing.T) {
	p := NewDefaultPolicy()
	if !p.Enabled {
		t.Fatal("expected enabled")
	}
	if p.MaxAge != 90*24*time.Hour {
		t.Fatalf("expected 90 days, got %v", p.MaxAge)
	}
	if p.MaxEvents != 0 {
		t.Fatal("expected 0 max events")
	}
}

// 8. TestNewDaysPolicy
func TestNewDaysPolicy(t *testing.T) {
	p := NewDaysPolicy(30)
	if p.MaxAge != 30*24*time.Hour {
		t.Fatalf("expected 30 days, got %v", p.MaxAge)
	}
}
