package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePercentage(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0}, {"0", 0}, {"50", 50}, {"100", 100}, {"150", 100}, {"-5", 0}, {"abc", 0},
	}
	for _, tt := range tests {
		if got := ParsePercentage(tt.input); got != tt.want {
			t.Errorf("ParsePercentage(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPickByPercentage(t *testing.T) {
	// 0% → never canary
	for i := uint64(0); i < 100; i++ {
		if pickByPercentage(0, i) {
			t.Fatal("0% should never route to canary")
		}
	}
	// 100% → always canary
	for i := uint64(0); i < 100; i++ {
		if !pickByPercentage(100, i) {
			t.Fatal("100% should always route to canary")
		}
	}
	// 50% → approximately half
	canary := 0
	total := 1000
	for i := uint64(1); i <= uint64(total); i++ {
		if pickByPercentage(50, i) {
			canary++
		}
	}
	if canary < 400 || canary > 600 {
		t.Errorf("50%% should route ~500/1000, got %d", canary)
	}
}

func TestCanaryRouter_HeaderOverride(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{
		"/api/v1/users": {Percentage: 0, Header: "X-Canary"},
	})
	cfg := cr.GetCanaryConfig("/api/v1/users")
	if cfg == nil {
		t.Fatal("expected config for /api/v1/users")
	}
	// Force canary via header
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("X-Canary", "true")
	if !cr.ShouldRouteCanary(cfg, req) {
		t.Error("header=true should route to canary")
	}
	// Force stable via header
	req2 := httptest.NewRequest("GET", "/api/v1/users", nil)
	req2.Header.Set("X-Canary", "false")
	if cr.ShouldRouteCanary(cfg, req2) {
		t.Error("header=false should NOT route to canary")
	}
}

func TestCanaryRouter_CookieOverride(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{
		"/api/v1/auth": {Percentage: 0, CookieName: "canary-sticky"},
	})
	cfg := cr.GetCanaryConfig("/api/v1/auth")
	// Cookie = canary → route to canary
	req := httptest.NewRequest("GET", "/api/v1/auth", nil)
	req.AddCookie(&http.Cookie{Name: "canary-sticky", Value: "canary"})
	if !cr.ShouldRouteCanary(cfg, req) {
		t.Error("cookie=canary should route to canary")
	}
	// Cookie = stable → don't route to canary
	req2 := httptest.NewRequest("GET", "/api/v1/auth", nil)
	req2.AddCookie(&http.Cookie{Name: "canary-sticky", Value: "stable"})
	if cr.ShouldRouteCanary(cfg, req2) {
		t.Error("cookie=stable should NOT route to canary")
	}
}

func TestCanaryRouter_PercentageZero(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{
		"/api/v1/users": {Percentage: 0},
	})
	cfg := cr.GetCanaryConfig("/api/v1/users")
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		if cr.ShouldRouteCanary(cfg, req) {
			t.Fatal("0% should never route to canary")
		}
	}
}

func TestCanaryRouter_PercentageFull(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{
		"/api/v1/users": {Percentage: 100},
	})
	cfg := cr.GetCanaryConfig("/api/v1/users")
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		if !cr.ShouldRouteCanary(cfg, req) {
			t.Fatal("100% should always route to canary")
		}
	}
}

func TestCanaryRouter_PercentageHalf(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{
		"/api/v1/audit": {Percentage: 50},
	})
	cfg := cr.GetCanaryConfig("/api/v1/audit")
	canary := 0
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/api/v1/audit", nil)
		if cr.ShouldRouteCanary(cfg, req) {
			canary++
		}
	}
	if canary < 30 || canary > 70 {
		t.Errorf("50%% should route ~50/100, got %d", canary)
	}
}

func TestGetCanaryConfig_NilRouter(t *testing.T) {
	var cr *CanaryRouter
	if cfg := cr.GetCanaryConfig("/api/v1/users"); cfg != nil {
		t.Error("nil router should return nil config")
	}
}

func TestGetCanaryConfig_NotFound(t *testing.T) {
	cr := NewCanaryRouter(map[string]*CanaryConfig{})
	if cfg := cr.GetCanaryConfig("/unknown"); cfg != nil {
		t.Error("unknown route should return nil config")
	}
}

func TestSetCanaryCookie(t *testing.T) {
	w := httptest.NewRecorder()
	SetCanaryCookie(w, "test-cookie", "canary")
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "test-cookie" || cookies[0].Value != "canary" {
		t.Errorf("unexpected cookie: %+v", cookies[0])
	}
}

func TestRandomBool(t *testing.T) {
	// 0% → always false
	for i := 0; i < 100; i++ {
		if RandomBool(0) {
			t.Fatal("0% should return false")
		}
	}
	// 100% → always true
	for i := 0; i < 100; i++ {
		if !RandomBool(100) {
			t.Fatal("100% should return true")
		}
	}
}
