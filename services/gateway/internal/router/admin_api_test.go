package router

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// adminAuthHeader is a Bearer token with admin scope for tests.
var adminAuthHeader = func() string {
	payload, _ := json.Marshal(map[string]any{
		"sub":    "admin-user",
		"scopes": []string{"platform:admin"},
	})
	return "Bearer eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}()

// adminRequest creates a request with admin JWT auth header.
func adminRequest(method, url string) *http.Request {
	r := httptest.NewRequest(method, url, nil)
	r.Header.Set("Authorization", adminAuthHeader)
	return r
}

func TestAdminRoutes_ListRoutes(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, adminRequest("GET", "/api/v1/admin/routes"))

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
	gw.ServeHTTP(w, adminRequest("GET", "/api/v1/admin/stats"))

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
	gw.ServeHTTP(w, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

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
	w1 := httptest.NewRecorder()
	gw.ServeHTTP(w1, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, adminRequest("POST", "/api/v1/admin/routes//api/v1/users/toggle"))

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
	gw.ServeHTTP(w, adminRequest("POST", "/api/v1/admin/routes//nonexistent/toggle"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAdminRoutes_ForbiddenWithoutAdminScope(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/admin/routes", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without admin scope, got %d", w.Code)
	}
}
