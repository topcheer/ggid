package router

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedAppRouter_AllowWithNoPolicy(t *testing.T) {
	router := NewProtectedAppRouter()
	router.RegisterApp(&ProtectedApp{
		Slug:        "grafana",
		UpstreamURL: "http://localhost:3000",
		AuthMode:    "jwt",
	})

	// Start a test upstream server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer upstream.Close()

	router.RegisterApp(&ProtectedApp{
		Slug:        "test-app",
		UpstreamURL: upstream.URL,
		AuthMode:    "jwt",
	})

	req := httptest.NewRequest("GET", "/app/test-app/dashboard", nil)
	w := httptest.NewRecorder()
	handled := router.HandleRequest(w, req)

	if !handled {
		t.Fatal("should handle /app/ path")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProtectedAppRouter_DenyOnPolicyFail(t *testing.T) {
	router := NewProtectedAppRouter()
	router.RegisterApp(&ProtectedApp{
		Slug:        "admin",
		UpstreamURL: "http://localhost:8080",
		AuthMode:    "jwt",
		AccessPolicy: map[string]any{
			"conditions": map[string]any{
				"and": []any{
					map[string]any{"$user.role": "admin"},
				},
			},
		},
	})

	req := httptest.NewRequest("GET", "/app/admin/config", nil)
	req.Header.Set("X-User-Role", "viewer") // not admin
	w := httptest.NewRecorder()

	router.HandleRequest(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 deny, got %d", w.Code)
	}
}

func TestProtectedAppRouter_StepupOnDeviceNotTrusted(t *testing.T) {
	router := NewProtectedAppRouter()
	router.RegisterApp(&ProtectedApp{
		Slug:        "secure",
		UpstreamURL: "http://localhost:9000",
		AuthMode:    "jwt",
		AccessPolicy: map[string]any{
			"conditions": map[string]any{
				"and": []any{
					map[string]any{"$security.device_trusted": true},
				},
			},
		},
	})

	req := httptest.NewRequest("GET", "/app/secure/data", nil)
	req.Header.Set("X-Device-Trusted", "false")
	w := httptest.NewRecorder()

	router.HandleRequest(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 stepup, got %d", w.Code)
	}
	if w.Header().Get("Require-MFA") != "true" {
		t.Error("should set Require-MFA header")
	}
}

func TestProtectedAppRouter_NotAnAppPath(t *testing.T) {
	router := NewProtectedAppRouter()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handled := router.HandleRequest(w, req)
	if handled {
		t.Fatal("should not handle non-/app/ paths")
	}
}

func TestProtectedAppRouter_HeaderInjection(t *testing.T) {
	router := NewProtectedAppRouter()

	// Start upstream that echoes headers.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Got-User", r.Header.Get("X-GGID-User"))
		w.Header().Set("X-Got-Roles", r.Header.Get("X-GGID-Roles"))
		w.Header().Set("X-Got-Tenant", r.Header.Get("X-GGID-Tenant"))
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	router.RegisterApp(&ProtectedApp{
		Slug:        "echo",
		UpstreamURL: upstream.URL,
		AuthMode:    "jwt",
		InjectHeaders: []map[string]any{
			{"name": "X-WebAuth-User", "value": "$user.email"},
		},
	})

	// Create a fake JWT with matching claims (ExtractJWTClaims parses without verification).
	jwtPayload := `{"sub":"user-123","tenant_id":"tenant-456","email":"alice@example.com","roles":["admin","sre"],"scopes":["openid"]}`
	jwtB64 := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	fakeJWT := "eyJhbGciOiJSUzI1NiJ9." + jwtB64 + ".fake-sig"

	req := httptest.NewRequest("GET", "/app/echo/", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT)
	// Attempt to forge headers — must be stripped.
	req.Header.Set("X-GGID-User", "forged")
	req.Header.Set("X-User-ID", "forged-user")

	w := httptest.NewRecorder()
	router.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Got-User") != "user-123" {
		t.Errorf("expected user-123, got %s", w.Header().Get("X-Got-User"))
	}
	if w.Header().Get("X-Got-Roles") != "admin,sre" {
		t.Errorf("expected roles, got %s", w.Header().Get("X-Got-Roles"))
	}
	if w.Header().Get("X-Got-Tenant") != "tenant-456" {
		t.Errorf("expected tenant-456, got %s", w.Header().Get("X-Got-Tenant"))
	}
}

func TestProtectedAppRouter_HotReload(t *testing.T) {
	router := NewProtectedAppRouter()

	// Register app.
	router.RegisterApp(&ProtectedApp{
		Slug:        "jenkins",
		UpstreamURL: "http://localhost:8080",
	})
	if _, ok := router.GetApp("jenkins"); !ok {
		t.Fatal("app should be registered")
	}

	// Unregister (hot reload: app deleted).
	router.UnregisterApp("jenkins")
	if _, ok := router.GetApp("jenkins"); ok {
		t.Fatal("app should be unregistered")
	}
}
