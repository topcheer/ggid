package middleware

import (
	"sync"
	"testing"
	"time"
)

func TestWSSessionRegistry_RegisterGet(t *testing.T) {
	r := NewWSSessionRegistry()
	s := &WSSession{ID: "s1", TenantID: "t1", UserID: "u1", StartedAt: time.Now()}
	r.Register(s)

	got, ok := r.Get("s1")
	if !ok {
		t.Fatal("expected session found")
	}
	if got.TenantID != "t1" {
		t.Errorf("expected t1, got %s", got.TenantID)
	}
}

func TestWSSessionRegistry_Unregister(t *testing.T) {
	r := NewWSSessionRegistry()
	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1"})
	r.Unregister("s1")

	if _, ok := r.Get("s1"); ok {
		t.Error("expected session removed")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0, got %d", r.Count())
	}
}

func TestWSSessionRegistry_Counts(t *testing.T) {
	r := NewWSSessionRegistry()
	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1"})
	r.Register(&WSSession{ID: "s2", TenantID: "t1", UserID: "u2"})
	r.Register(&WSSession{ID: "s3", TenantID: "t2", UserID: "u1"})

	if r.Count() != 3 {
		t.Errorf("expected 3 total, got %d", r.Count())
	}
	if r.CountByTenant("t1") != 2 {
		t.Errorf("expected 2 for t1, got %d", r.CountByTenant("t1"))
	}
	if r.CountByUser("u1") != 2 {
		t.Errorf("expected 2 for u1, got %d", r.CountByUser("u1"))
	}
}

func TestWSSessionRegistry_UnregisterCleansIndexes(t *testing.T) {
	r := NewWSSessionRegistry()
	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1"})
	r.Unregister("s1")

	if r.CountByTenant("t1") != 0 {
		t.Error("tenant index should be empty")
	}
	if r.CountByUser("u1") != 0 {
		t.Error("user index should be empty")
	}
}

func TestWSSessionRegistry_UnregisterUnknown(t *testing.T) {
	r := NewWSSessionRegistry()
	r.Unregister("nonexistent") // should not panic
}

func TestWSSessionRegistry_BroadcastToTenant(t *testing.T) {
	r := NewWSSessionRegistry()
	var mu sync.Mutex
	delivered := 0
	cb := func(msg []byte) {
		mu.Lock()
		delivered++
		mu.Unlock()
	}

	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1", OnMessage: cb})
	r.Register(&WSSession{ID: "s2", TenantID: "t1", UserID: "u2", OnMessage: cb})
	r.Register(&WSSession{ID: "s3", TenantID: "t2", UserID: "u3", OnMessage: cb})

	n := r.BroadcastToTenant("t1", []byte("hello"))
	if n != 2 {
		t.Errorf("expected 2 delivered, got %d", n)
	}
	mu.Lock()
	if delivered != 2 {
		t.Errorf("expected 2 callbacks, got %d", delivered)
	}
	mu.Unlock()
}

func TestWSSessionRegistry_BroadcastToUnknownTenant(t *testing.T) {
	r := NewWSSessionRegistry()
	n := r.BroadcastToTenant("nonexistent", []byte("x"))
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestWSSessionRegistry_SendToUser(t *testing.T) {
	r := NewWSSessionRegistry()
	called := false
	cb := func(msg []byte) { called = true }

	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1", OnMessage: cb})
	r.Register(&WSSession{ID: "s2", TenantID: "t1", UserID: "u1", OnMessage: cb})

	n := r.SendToUser("u1", []byte("hello"))
	if n != 2 {
		t.Errorf("expected 2 delivered, got %d", n)
	}
	if !called {
		t.Error("expected callback fired")
	}
}

func TestWSSessionRegistry_ListSessions(t *testing.T) {
	r := NewWSSessionRegistry()
	r.Register(&WSSession{ID: "s1", TenantID: "t1", UserID: "u1", RemoteAddr: "1.2.3.4"})
	r.Register(&WSSession{ID: "s2", TenantID: "t2", UserID: "u2", RemoteAddr: "5.6.7.8"})

	list := r.ListSessions()
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
	// Verify no callbacks leaked
	for _, info := range list {
		if info.ID == "" {
			t.Error("expected non-empty ID")
		}
	}
}
