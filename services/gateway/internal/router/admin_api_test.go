package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminRoutes_ListRoutes(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/admin/routes", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	routes, ok := resp["routes"].([]any)
	if !ok {
		t.Fatal("expected routes array")
	}
	if len(routes) == 0 {
		t.Error("expected at least 1 route")
	}
}

func TestAdminStats_ReturnsBackends(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp AdminStatsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Backends) == 0 {
		t.Error("expected at least 1 backend")
	}
}

func TestAdminToggleRoute_Disable(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", resp["enabled"])
	}
}

func TestAdminToggleRoute_Enable(t *testing.T) {
	gw := newTestGateway(t)

	// First disable
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle", nil)
	gw.ServeHTTP(w1, r1)

	// Then re-enable
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle", nil)
	gw.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", resp["enabled"])
	}
}

func TestAdminToggleRoute_NotFound(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/admin/routes//nonexistent/toggle", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// helper to read body from recorder
func init() {}
