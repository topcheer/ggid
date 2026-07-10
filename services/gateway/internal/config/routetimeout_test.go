package config

import (
	"testing"
	"time"
)

func TestGetRouteTimeout_WithExplicitConfig(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/auth": {
				Timeout: RouteTimeout{
					Read:  5 * time.Second,
					Write: 10 * time.Second,
					Idle:  60 * time.Second,
					Dial:  3 * time.Second,
				},
			},
		},
	}
	to := cfg.GetRouteTimeout("/api/v1/auth")
	if to.Read != 5*time.Second {
		t.Errorf("expected Read=5s, got %v", to.Read)
	}
	if to.Write != 10*time.Second {
		t.Errorf("expected Write=10s, got %v", to.Write)
	}
	if to.Idle != 60*time.Second {
		t.Errorf("expected Idle=60s, got %v", to.Idle)
	}
	if to.Dial != 3*time.Second {
		t.Errorf("expected Dial=3s, got %v", to.Dial)
	}
}

func TestGetRouteTimeout_FallbackToDefaults(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 25 * time.Second,
	}
	to := cfg.GetRouteTimeout("/unknown")
	if to.Read != 20*time.Second {
		t.Errorf("expected default Read=20s, got %v", to.Read)
	}
	if to.Write != 25*time.Second {
		t.Errorf("expected default Write=25s, got %v", to.Write)
	}
	// Default idle and dial
	if to.Idle != 90*time.Second {
		t.Errorf("expected default Idle=90s, got %v", to.Idle)
	}
	if to.Dial != 5*time.Second {
		t.Errorf("expected default Dial=5s, got %v", to.Dial)
	}
}

func TestGetRouteTimeout_PartialFallback(t *testing.T) {
	cfg := &Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		RouteConfigs: map[string]RouteConfig{
			"/api/v1/custom": {
				Timeout: RouteTimeout{
					Read: 8 * time.Second,
					// Write, Idle, Dial are zero → should fall back
				},
			},
		},
	}
	to := cfg.GetRouteTimeout("/api/v1/custom")
	if to.Read != 8*time.Second {
		t.Errorf("expected explicit Read=8s, got %v", to.Read)
	}
	if to.Write != 15*time.Second {
		t.Errorf("expected fallback Write=15s, got %v", to.Write)
	}
	if to.Idle != 90*time.Second {
		t.Errorf("expected fallback Idle=90s, got %v", to.Idle)
	}
	if to.Dial != 5*time.Second {
		t.Errorf("expected fallback Dial=5s, got %v", to.Dial)
	}
}

func TestDefault_HasRouteConfigs(t *testing.T) {
	cfg := Default()
	if len(cfg.RouteConfigs) == 0 {
		t.Error("Default() should have at least one RouteConfig")
	}
	authTO, ok := cfg.RouteConfigs["/api/v1/auth"]
	if !ok {
		t.Error("expected RouteConfig for /api/v1/auth")
	}
	if authTO.Timeout.Read == 0 {
		t.Error("expected non-zero Read timeout for /api/v1/auth")
	}
}
