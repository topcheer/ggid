package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- wsproxy.go: IsWebSocketRequest ---

func TestIsWebSocketRequest_ValidUpgrade_V2(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	if !IsWebSocketRequest(req) {
		t.Error("expected true for valid WS upgrade")
	}
}

func TestIsWebSocketRequest_NoUpgrade_V2(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	if IsWebSocketRequest(req) {
		t.Error("expected false for non-WS request")
	}
}

func TestIsWebSocketRequest_PartialUpgrade_V2(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	if IsWebSocketRequest(req) {
		t.Error("expected false when Connection header missing")
	}
}

// --- ws_registry.go ---

func TestWSSessionRegistry_RegisterAndGet_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	sess := &WSSession{
		ID: "s1", TenantID: "t1", UserID: "u1",
		StartedAt: time.Now(), RemoteAddr: "10.0.0.1:1234",
		OnMessage: func(msg []byte) {},
	}
	reg.Register(sess)
	if reg.Count() != 1 {
		t.Errorf("expected 1, got %d", reg.Count())
	}
	got, ok := reg.Get("s1")
	if !ok || got.ID != "s1" {
		t.Error("expected to find session s1")
	}
}

func TestWSSessionRegistry_Unregister_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Register(&WSSession{ID: "s2", TenantID: "t1", UserID: "u1", StartedAt: time.Now()})
	reg.Unregister("s2")
	if reg.Count() != 0 {
		t.Error("expected 0 after unregister")
	}
}

func TestWSSessionRegistry_BroadcastToTenant_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	var received []byte
	reg.Register(&WSSession{
		ID: "s3", TenantID: "t1", UserID: "u1", StartedAt: time.Now(),
		OnMessage: func(msg []byte) { received = msg },
	})
	n := reg.BroadcastToTenant("t1", []byte("hello"))
	if n != 1 {
		t.Errorf("expected 1 delivered, got %d", n)
	}
	if string(received) != "hello" {
		t.Errorf("expected 'hello', got %s", string(received))
	}
}

func TestWSSessionRegistry_BroadcastWrongTenant_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Register(&WSSession{
		ID: "s4", TenantID: "t1", UserID: "u1", StartedAt: time.Now(),
		OnMessage: func(msg []byte) {},
	})
	n := reg.BroadcastToTenant("wrong", []byte("msg"))
	if n != 0 {
		t.Errorf("expected 0 delivered, got %d", n)
	}
}

func TestWSSessionRegistry_SendToUser_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	var received []byte
	reg.Register(&WSSession{
		ID: "s5", TenantID: "t1", UserID: "u1", StartedAt: time.Now(),
		OnMessage: func(msg []byte) { received = msg },
	})
	n := reg.SendToUser("u1", []byte("user-msg"))
	if n != 1 {
		t.Errorf("expected 1 delivered, got %d", n)
	}
	if string(received) != "user-msg" {
		t.Error("expected user-msg")
	}
}

func TestWSSessionRegistry_SendWrongUser_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Register(&WSSession{
		ID: "s6", TenantID: "t1", UserID: "u1", StartedAt: time.Now(),
		OnMessage: func(msg []byte) {},
	})
	n := reg.SendToUser("wrong", []byte("msg"))
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestWSSessionRegistry_CountByTenant_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	for i := 0; i < 3; i++ {
		reg.Register(&WSSession{
			ID: "s" + string(rune('a'+i)), TenantID: "shared",
			UserID: "u", StartedAt: time.Now(),
		})
	}
	if reg.CountByTenant("shared") != 3 {
		t.Errorf("expected 3, got %d", reg.CountByTenant("shared"))
	}
}

func TestWSSessionRegistry_CountByUser_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Register(&WSSession{ID: "s7", TenantID: "t1", UserID: "target", StartedAt: time.Now()})
	reg.Register(&WSSession{ID: "s8", TenantID: "t1", UserID: "target", StartedAt: time.Now()})
	if reg.CountByUser("target") != 2 {
		t.Errorf("expected 2, got %d", reg.CountByUser("target"))
	}
}

func TestWSSessionRegistry_ListSessions_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Register(&WSSession{ID: "s9", TenantID: "t1", UserID: "u1", StartedAt: time.Now()})
	reg.Register(&WSSession{ID: "s10", TenantID: "t2", UserID: "u2", StartedAt: time.Now()})
	list := reg.ListSessions()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestWSSessionRegistry_UnregisterNonExistent_V2(t *testing.T) {
	reg := NewWSSessionRegistry()
	reg.Unregister("nonexistent")
	if reg.Count() != 0 {
		t.Error("expected 0")
	}
}
