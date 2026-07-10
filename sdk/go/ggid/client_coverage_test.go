package ggid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Client creation tests ---

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient("http://localhost:8080")
	if c.gatewayURL != "http://localhost:8080" {
		t.Errorf("gatewayURL = %q", c.gatewayURL)
	}
	if c.tenantID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("default tenantID = %q", c.tenantID)
	}
	if c.httpClient == nil {
		t.Error("httpClient should be non-nil")
	}
	if c.httpClient.Timeout != 30000000000 {
		t.Errorf("timeout = %v", c.httpClient.Timeout)
	}
}

func TestWithTenantID(t *testing.T) {
	c := NewClient("http://localhost:8080", WithTenantID("my-tenant"))
	if c.tenantID != "my-tenant" {
		t.Errorf("tenantID = %q, want 'my-tenant'", c.tenantID)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5000000000}
	c := NewClient("http://localhost:8080", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("httpClient should be the custom client")
	}
}

func TestWithJWKS(t *testing.T) {
	c := NewClient("http://localhost:8080", WithJWKS("http://localhost:8080/.well-known/jwks.json"))
	if c.verifier == nil {
		t.Error("verifier should be non-nil")
	}
}

func TestWithCredentials(t *testing.T) {
	c := NewClient("http://localhost:8080", WithCredentials("admin", "pass"))
	if c.username != "admin" || c.password != "pass" {
		t.Errorf("credentials = %s/%s", c.username, c.password)
	}
}

// --- ensureContext ---

func TestEnsureContext(t *testing.T) {
	ctx := context.Background()
	if ensureContext(ctx) == nil {
		t.Error("non-nil context should be returned")
	}
	if ensureContext(nil) == nil {
		t.Error("nil context should return Background")
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenSet{
			AccessToken: "access-123",
			RefreshToken: "refresh-456",
			TokenType: "Bearer",
			ExpiresIn: 3600,
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	tokens, err := c.Login(context.Background(), "admin", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "access-123" {
		t.Errorf("access_token = %q", tokens.AccessToken)
	}
	if tokens.RefreshToken != "refresh-456" {
		t.Errorf("refresh_token = %q", tokens.RefreshToken)
	}
}

func TestLogin_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.Login(context.Background(), "admin", "wrong")
	if err == nil {
		t.Error("expected error on 401")
	}
}

func TestLogin_NilContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(TokenSet{AccessToken: "tok"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	tokens, err := c.Login(nil, "admin", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "tok" {
		t.Errorf("access_token = %q", tokens.AccessToken)
	}
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"user_id": "u123"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	id, err := c.Register(context.Background(), "user1", "user@example.com", "pass", "User One")
	if err != nil {
		t.Fatal(err)
	}
	if id != "u123" {
		t.Errorf("user_id = %q", id)
	}
}

func TestRegister_EmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	id, err := c.Register(context.Background(), "user1", "user@example.com", "pass", "User One")
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Errorf("expected empty user_id, got %q", id)
	}
}

// --- Refresh ---

func TestRefresh_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(TokenSet{AccessToken: "new-access", ExpiresIn: 3600})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	tokens, err := c.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "new-access" {
		t.Errorf("access_token = %q", tokens.AccessToken)
	}
}

func TestRefresh_InvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.Refresh(context.Background(), "old-refresh")
	if err == nil {
		t.Error("expected parse error")
	}
}

// --- ListUsers ---

func TestListUsers_NestedFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"users": []User{
				{ID: "1", Username: "alice"},
				{ID: "2", Username: "bob"},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	users, err := c.ListUsers(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("len = %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("username[0] = %q", users[0].Username)
	}
}

func TestListUsers_FlatArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]User{
			{ID: "1", Username: "alice"},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	users, err := c.ListUsers(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("len = %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("username = %q", users[0].Username)
	}
}

func TestListUsers_InvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.ListUsers(context.Background(), "token")
	if err == nil {
		t.Error("expected error")
	}
}

func TestListUsers_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.ListUsers(context.Background(), "token")
	if err == nil {
		t.Error("expected error on 500")
	}
}

// --- GetUser ---

func TestGetUser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(User{ID: "u1", Username: "alice", Email: "alice@example.com"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	user, err := c.GetUser(context.Background(), "token", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "alice" {
		t.Errorf("username = %q", user.Username)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.GetUser(context.Background(), "token", "nonexistent")
	if err == nil {
		t.Error("expected error on 404")
	}
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	if err := c.DeleteUser(context.Background(), "token", "u1"); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteUser_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	if err := c.DeleteUser(context.Background(), "token", "u1"); err == nil {
		t.Error("expected error on 500")
	}
}

// --- ListRoles ---

func TestListRoles_NestedFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"roles": []Role{
				{ID: "r1", Name: "Admin", Key: "admin", SystemRole: true},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	roles, err := c.ListRoles(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("len = %d", len(roles))
	}
	if roles[0].Name != "Admin" {
		t.Errorf("name = %q", roles[0].Name)
	}
}

func TestListRoles_FlatArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Role{{ID: "r1", Name: "Admin"}})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	roles, err := c.ListRoles(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("len = %d", len(roles))
	}
}

func TestListRoles_InvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid"))
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.ListRoles(context.Background(), "token")
	if err == nil {
		t.Error("expected error")
	}
}

// --- CheckPermission ---

func TestCheckPermission_Allowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PolicyResult{Allowed: true, Reason: "permitted"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	result, err := c.CheckPermission(context.Background(), "token", "docs", "read")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Error("expected allowed")
	}
}

func TestCheckPermission_Denied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PolicyResult{Allowed: false, Reason: "insufficient permissions"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	result, err := c.CheckPermission(context.Background(), "token", "docs", "delete")
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed {
		t.Error("expected denied")
	}
}

func TestCheckPermission_InvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid"))
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.CheckPermission(context.Background(), "token", "docs", "read")
	if err == nil {
		t.Error("expected error")
	}
}

// --- VerifyToken ---

func TestVerifyToken_NoJWKSConfigured(t *testing.T) {
	c := NewClient("http://localhost:8080")
	_, err := c.VerifyToken(context.Background(), "some.jwt.token")
	if err == nil {
		t.Error("expected error when no JWKS configured")
	}
}

// --- Internal HTTP (do) ---

func TestDo_SetsTenantHeader(t *testing.T) {
	var capturedTenant string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get("X-Tenant-ID")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := NewClient(ts.URL, WithTenantID("custom-tenant"))
	c.do(context.Background(), http.MethodGet, "/test", nil, "")

	if capturedTenant != "custom-tenant" {
		t.Errorf("X-Tenant-ID = %q", capturedTenant)
	}
}

func TestDo_SetsBearerToken(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.do(context.Background(), http.MethodGet, "/test", nil, "my-token")

	if capturedAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q", capturedAuth)
	}
}

func TestDo_SetsContentType(t *testing.T) {
	var capturedCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCT = r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.do(context.Background(), http.MethodGet, "/test", nil, "")

	if capturedCT != "application/json" {
		t.Errorf("Content-Type = %q", capturedCT)
	}
}

func TestDo_RequestFailed(t *testing.T) {
	c := NewClient("http://127.0.0.1:0") // invalid port
	_, err := c.do(context.Background(), http.MethodGet, "/test", nil, "")
	if err == nil {
		t.Error("expected error on failed connection")
	}
}

func TestDo_NilContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	_, err := c.do(nil, http.MethodGet, "/test", nil, "")
	if err != nil {
		t.Fatal(err)
	}
}

// --- RequirePermission ---

func TestRequirePermission_Allowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PolicyResult{Allowed: true})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// Need claims in context (set by Middleware)
	handler := c.Middleware(c.RequirePermission("docs", "read")(inner))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	// We need a valid-looking JWT to pass middleware
	// Create a simple JWT that will parse correctly
	payload := `{"sub":"test","exp":9999999999}`
	encodedPayload := base64URLEncode([]byte(payload))
	token := "header." + encodedPayload + ".signature"
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PolicyResult{Allowed: false})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	handler := c.Middleware(c.RequirePermission("docs", "delete")(inner))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	payload := `{"sub":"test","exp":9999999999}`
	encodedPayload := base64URLEncode([]byte(payload))
	token := "header." + encodedPayload + ".signature"
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	c := NewClient("http://localhost:8080")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	})

	handler := c.Middleware(next)

	// Create expired JWT
	payload := `{"sub":"test","exp":1000}`
	encodedPayload := base64URLEncode([]byte(payload))
	token := "header." + encodedPayload + ".signature"
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_DocsPathSkipped(t *testing.T) {
	c := NewClient("http://localhost:8080")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := c.Middleware(next)
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("/docs should be public")
	}
}

func TestMiddleware_ApiDocsPathSkipped(t *testing.T) {
	c := NewClient("http://localhost:8080")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := c.Middleware(next)
	req := httptest.NewRequest("GET", "/api-docs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("/api-docs should be public")
	}
}

// --- Errors ---

func TestErrors(t *testing.T) {
	if ErrNotAuthenticated == nil {
		t.Error("ErrNotAuthenticated should be non-nil")
	}
	if ErrTokenExpired == nil {
		t.Error("ErrTokenExpired should be non-nil")
	}
}

// --- Helpers ---

func base64URLEncode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var result []byte
	for i := 0; i < len(data); i += 3 {
		b1 := data[i]
		var b2, b3 byte
		if i+1 < len(data) { b2 = data[i+1] }
		if i+2 < len(data) { b3 = data[i+2] }

		result = append(result, alphabet[b1>>2])
		result = append(result, alphabet[((b1&0x03)<<4)|(b2>>4)])
		if i+1 < len(data) {
			result = append(result, alphabet[((b2&0x0f)<<2)|(b3>>6)])
		} else {
			result = append(result, '=')
			result = append(result, '=')
			break
		}
		if i+2 < len(data) {
			result = append(result, alphabet[b3&0x3f])
		} else {
			result = append(result, '=')
			break
		}
	}
	return string(result)
}
