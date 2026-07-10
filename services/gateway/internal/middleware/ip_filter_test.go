package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPFilter_AllowByDefault(t *testing.T) {
	store := NewIPFilterStore(nil)
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestIPFilter_Disabled(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: false, DenyList: []string{"0.0.0.0/0"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("disabled filter: want 200, got %d", rr.Code)
	}
}

func TestIPFilter_DenyList(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: true, DenyList: []string{"192.168.1.0/24"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Errorf("denied IP: want 403, got %d", rr.Code)
	}
}

func TestIPFilter_DenyListAllowed(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: true, DenyList: []string{"192.168.1.0/24"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("non-denied IP: want 200, got %d", rr.Code)
	}
}

func TestIPFilter_AllowList(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: true, AllowList: []string{"10.0.0.0/8"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// IP in allowlist
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("allowed IP: want 200, got %d", rr.Code)
	}
	// IP not in allowlist
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 403 {
		t.Errorf("non-allowed IP: want 403, got %d", rr2.Code)
	}
}

func TestIPFilter_HealthCheckSkipped(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: true, DenyList: []string{"0.0.0.0/0"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("healthz: want 200, got %d", rr.Code)
	}
}

func TestIPFilter_PerTenant(t *testing.T) {
	store := NewIPFilterStore(nil)
	store.Set("t1", &IPFilterConfig{Enabled: true, AllowList: []string{"10.0.0.0/8"}})

	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// Tenant t1 with allowed IP
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "t1")
	req = req.WithContext(ctx)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("t1 allowed IP: want 200, got %d", rr.Code)
	}
	// Tenant t1 with non-allowed IP
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := context.WithValue(req2.Context(), TenantIDKey, "t1")
	req2 = req2.WithContext(ctx2)
	req2.RemoteAddr = "192.168.1.1:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 403 {
		t.Errorf("t1 non-allowed IP: want 403, got %d", rr2.Code)
	}
}

func TestIPFilter_SingleIP(t *testing.T) {
	store := NewIPFilterStore(&IPFilterConfig{Enabled: true, DenyList: []string{"1.2.3.4"}})
	handler := IPFilterMiddleware(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Errorf("single IP deny: want 403, got %d", rr.Code)
	}
}

func TestIPFilterStore_SetGetDelete(t *testing.T) {
	store := NewIPFilterStore(nil)
	cfg := &IPFilterConfig{Enabled: true, DenyList: []string{"0.0.0.0/0"}}
	store.Set("t1", cfg)
	if store.Get("t1") != cfg {
		t.Error("Get should return same config")
	}
	store.Delete("t1")
	if store.Get("t1") != nil {
		t.Error("Get after delete should return nil")
	}
}

func TestParseCIDRList(t *testing.T) {
	nets := parseCIDRList([]string{"10.0.0.0/8", "192.168.0.0/16"})
	if len(nets) != 2 {
		t.Errorf("want 2 networks, got %d", len(nets))
	}
}

func TestParseCIDRList_SingleIP(t *testing.T) {
	nets := parseCIDRList([]string{"1.2.3.4"})
	if len(nets) != 1 {
		t.Errorf("want 1 network, got %d", len(nets))
	}
}

func TestIPInList(t *testing.T) {
	nets := parseCIDRList([]string{"10.0.0.0/8"})
	if !ipInList("10.1.2.3", nets) {
		t.Error("10.1.2.3 should be in 10.0.0.0/8")
	}
	if ipInList("192.168.1.1", nets) {
		t.Error("192.168.1.1 should not be in 10.0.0.0/8")
	}
}

func TestIPInList_InvalidIP(t *testing.T) {
	nets := parseCIDRList([]string{"10.0.0.0/8"})
	if ipInList("invalid-ip", nets) {
		t.Error("Invalid IP should return false")
	}
}
