package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostValidation_AllowExact_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts: []string{"api.example.com", "console.example.com"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Allowed host
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com"
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 for allowed host, got %d", rr.Code)
	}
}

func TestHostValidation_Disallowed_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts: []string{"api.example.com"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Disallowed host → 403
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "evil.attacker.com"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestHostValidation_EmptyAllowlist_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// No allowlist = allow all
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "anything.com"
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 with no allowlist, got %d", rr.Code)
	}
}

func TestHostValidation_PortStripping_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts:      []string{"api.example.com"},
		AllowPortStripping: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Host with port that matches after stripping
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com:8080"
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 with port stripping, got %d", rr.Code)
	}
}

func TestHostValidation_PortNotAllowed_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts:      []string{"api.example.com"},
		AllowPortStripping: true,
		AllowedPorts:       []string{"443"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Port 8080 not in allowed ports
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com:8080"
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disallowed port, got %d", rr.Code)
	}
}

func TestHostValidation_PortAllowed_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts:      []string{"api.example.com"},
		AllowPortStripping: true,
		AllowedPorts:       []string{"443", "8080"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Port 8080 in allowed ports
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com:8080"
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 for allowed port, got %d", rr.Code)
	}
}

func TestHostValidation_CaseInsensitive_C22(t *testing.T) {
	h := HostValidation(HostValidationConfig{
		AllowedHosts: []string{"API.Example.COM"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Lowercase host should match uppercase config
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com"
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expected 200 case-insensitive match, got %d", rr.Code)
	}
}
