package router

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
)

func TestGateway_GetRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/test": "http://localhost:9999",
		"/api/v1/auth": "http://localhost:9001",
	}
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/gateway/routes", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp RoutesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(resp.Routes))
	}
}

func TestGateway_GetRoutes_NotFound(t *testing.T) {
	gw := testGatewayNoJWKS(t)

	// GET to unknown gateway path should return 404
	req := httptest.NewRequest("GET", "/api/v1/gateway/unknown", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGateway_ReloadRoutes_NoReloadFunc(t *testing.T) {
	gw := testGatewayNoJWKS(t)

	req := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("expected 503 without reload func, got %d", w.Code)
	}
}

func TestGateway_ReloadRoutes_WithReloadFunc(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": "http://localhost:9999"}
	gw := New(cfg, nil)

	callCount := 0
	gw.SetReloadFunc(func() (*config.Config, error) {
		callCount++
		newCfg := config.Default()
		newCfg.Routes = map[string]string{
			"/api/v1/test":  "http://localhost:9999",
			"/api/v1/new":   "http://localhost:8888",
		}
		return newCfg, nil
	})

	// Before reload
	req1 := httptest.NewRequest("GET", "/api/v1/gateway/routes", nil)
	w1 := httptest.NewRecorder()
	gw.ServeHTTP(w1, req1)
	var resp1 RoutesResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	if len(resp1.Routes) != 1 {
		t.Fatalf("expected 1 route before reload, got %d", len(resp1.Routes))
	}

	// Reload
	req2 := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 on reload, got %d", w2.Code)
	}
	if callCount != 1 {
		t.Error("reload func should be called once")
	}

	// After reload
	req3 := httptest.NewRequest("GET", "/api/v1/gateway/routes", nil)
	w3 := httptest.NewRecorder()
	gw.ServeHTTP(w3, req3)
	var resp3 RoutesResponse
	json.NewDecoder(w3.Body).Decode(&resp3)
	if len(resp3.Routes) != 2 {
		t.Errorf("expected 2 routes after reload, got %d", len(resp3.Routes))
	}
}

func TestGateway_ReloadRoutes_ReloadFuncError(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	gw.SetReloadFunc(func() (*config.Config, error) {
		return nil, fmt.Errorf("config file not found")
	})

	req := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500 on reload error, got %d", w.Code)
	}
}

func TestGateway_SetReloadFunc(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	fn := func() (*config.Config, error) { return config.Default(), nil }
	gw.SetReloadFunc(fn)
	if gw.reloadFunc == nil {
		t.Error("expected reloadFunc to be set")
	}
}

func TestGateway_RouteVersionIncrements(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": "http://localhost:9999"}
	gw := New(cfg, nil)
	gw.SetReloadFunc(func() (*config.Config, error) {
		return cfg, nil
	})

	// Get initial version
	req1 := httptest.NewRequest("GET", "/api/v1/gateway/routes", nil)
	w1 := httptest.NewRecorder()
	gw.ServeHTTP(w1, req1)
	var resp1 RoutesResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	v1 := resp1.Version

	// Reload
	req2 := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, req2)

	// Check version incremented
	req3 := httptest.NewRequest("GET", "/api/v1/gateway/routes", nil)
	w3 := httptest.NewRecorder()
	gw.ServeHTTP(w3, req3)
	var resp3 RoutesResponse
	json.NewDecoder(w3.Body).Decode(&resp3)
	if resp3.Version <= v1 {
		t.Errorf("expected version to increment, got %d -> %d", v1, resp3.Version)
	}
}
