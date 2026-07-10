package ggid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Test helpers ---

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, WithHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	return c, srv
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- Auth tests ---

func TestLogin(t *testing.T) {
	expectedToken := "test-access-token"
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "admin" || body["password"] != "pass" {
			t.Errorf("unexpected body: %v", body)
		}
		writeJSON(w, 200, TokenSet{
			AccessToken:  expectedToken,
			RefreshToken: "refresh",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))

	ts, err := c.Login(context.Background(), &LoginRequest{Username: "admin", Password: "pass"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if ts.AccessToken != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, ts.AccessToken)
	}
	if ts.ExpiresIn != 3600 {
		t.Errorf("expected expires_in 3600, got %d", ts.ExpiresIn)
	}
}

func TestRefreshToken(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["refresh_token"] != "rt-123" {
			t.Errorf("unexpected refresh_token: %v", body)
		}
		writeJSON(w, 200, TokenSet{AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 3600, TokenType: "Bearer"})
	}))

	ts, err := c.RefreshToken(context.Background(), "rt-123")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if ts.AccessToken != "new-access" {
		t.Errorf("expected new-access, got %s", ts.AccessToken)
	}
}

func TestLogout(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/logout" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))

	if err := c.Logout(context.Background(), "access-token"); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
}

// --- Token verification tests ---

func TestVerifyTokenOffline(t *testing.T) {
	// Create an unsigned JWT-like string for offline parsing.
	// Header: {"alg":"none","typ":"JWT"}, Payload with sub/email/roles/scope
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"sub":"user-1","username":"admin","email":"admin@test.com","tenant_id":"t-1","roles":["admin","editor"],"scope":"read write"}`
	sig := ""
	token := fmt.Sprintf("%s.%s.%s",
		base64URLEncode([]byte(header)),
		base64URLEncode([]byte(payload)),
		sig)

	c := New("http://localhost")
	info, err := c.VerifyToken(context.Background(), token)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if info.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", info.UserID)
	}
	if info.Email != "admin@test.com" {
		t.Errorf("expected admin@test.com, got %s", info.Email)
	}
	if info.TenantID != "t-1" {
		t.Errorf("expected tenant t-1, got %s", info.TenantID)
	}
	if len(info.Roles) != 2 || info.Roles[0] != "admin" {
		t.Errorf("unexpected roles: %v", info.Roles)
	}
	if len(info.Scopes) != 2 || info.Scopes[0] != "read" {
		t.Errorf("unexpected scopes: %v", info.Scopes)
	}
}

func TestVerifyTokenInvalid(t *testing.T) {
	c := New("http://localhost")
	_, err := c.VerifyToken(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func base64URLEncode(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	buf := make([]byte, 0, len(data)*4/3)
	for i := 0; i < len(data); i += 3 {
		var n uint32
		var cnt int
		for j := 0; j < 3 && i+j < len(data); j++ {
			n = n<<8 | uint32(data[i+j])
			cnt++
		}
		n <<= uint((3 - cnt) * 8)
		buf = append(buf, chars[(n>>18)&0x3F])
		buf = append(buf, chars[(n>>12)&0x3F])
		if cnt > 1 {
			buf = append(buf, chars[(n>>6)&0x3F])
		}
		if cnt > 2 {
			buf = append(buf, chars[n&0x3F])
		}
	}
	return string(buf)
}

// --- User management tests ---

func TestCreateUser(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var body CreateUserRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Username != "newuser" {
			t.Errorf("unexpected username: %s", body.Username)
		}
		writeJSON(w, 201, User{ID: "u-1", Username: body.Username, Email: body.Email})
	}))

	user, err := c.CreateUser(context.Background(), &CreateUserRequest{
		Username: "newuser",
		Email:    "new@test.com",
		Password: "Pass@123",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID != "u-1" {
		t.Errorf("expected u-1, got %s", user.ID)
	}
}

func TestGetUser(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/u-1" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, 200, User{ID: "u-1", Username: "testuser"})
	}))

	user, err := c.GetUser(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected testuser, got %s", user.Username)
	}
}

func TestUpdateUser(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		writeJSON(w, 200, User{ID: "u-1", Email: "updated@test.com"})
	}))

	newEmail := "updated@test.com"
	user, err := c.UpdateUser(context.Background(), "u-1", &UpdateUserRequest{Email: &newEmail})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if user.Email != "updated@test.com" {
		t.Errorf("expected updated email, got %s", user.Email)
	}
}

func TestDeleteUser(t *testing.T) {
	deleted := false
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/users/u-1" {
			deleted = true
			w.WriteHeader(204)
			return
		}
		t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
	}))

	if err := c.DeleteUser(context.Background(), "u-1"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if !deleted {
		t.Error("delete was not called")
	}
}

func TestListUsers(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		// Check pagination params.
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("page_size") != "20" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		writeJSON(w, 200, PageResult[User]{
			Items:      []User{{ID: "u-1"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		})
	}))

	result, err := c.ListUsers(context.Background(), &ListOptions{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("expected 1 total, got %d", result.TotalCount)
	}
}

func TestListUsersNilOpts(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PageResult[User]{Items: []User{}, TotalCount: 0})
	}))

	// nil opts should not panic.
	result, err := c.ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListUsers with nil opts failed: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected 0, got %d", result.TotalCount)
	}
}

func TestAssignRole(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/u-1/roles" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	}))

	if err := c.AssignRole(context.Background(), "u-1", "r-1"); err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}
}

func TestRemoveRole(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/u-1/roles/r-1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	}))

	if err := c.RemoveRole(context.Background(), "u-1", "r-1"); err != nil {
		t.Fatalf("RemoveRole failed: %v", err)
	}
}

// --- Role management tests ---

func TestCreateRole(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/roles" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, 201, Role{ID: "r-1", Key: "admin", Name: "Administrator"})
	}))

	role, err := c.CreateRole(context.Background(), &CreateRoleRequest{Key: "admin", Name: "Administrator"})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if role.Key != "admin" {
		t.Errorf("expected key admin, got %s", role.Key)
	}
}

func TestListRoles(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PageResult[Role]{
			Items:      []Role{{ID: "r-1", Key: "admin"}},
			TotalCount: 1,
		})
	}))

	result, err := c.ListRoles(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 role, got %d", len(result.Items))
	}
}

// --- Organization tests ---

func TestCreateOrg(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/organizations" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, 201, Organization{ID: "o-1", Name: "Engineering"})
	}))

	org, err := c.CreateOrg(context.Background(), &CreateOrgRequest{Name: "Engineering"})
	if err != nil {
		t.Fatalf("CreateOrg failed: %v", err)
	}
	if org.Name != "Engineering" {
		t.Errorf("expected Engineering, got %s", org.Name)
	}
}

func TestListOrgs(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, PageResult[Organization]{
			Items:      []Organization{{ID: "o-1", Name: "Engineering"}},
			TotalCount: 1,
		})
	}))

	result, err := c.ListOrgs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListOrgs failed: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("expected 1, got %d", result.TotalCount)
	}
}

// --- Permission check test ---

func TestCheckPermission(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/check" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, 200, map[string]any{"allowed": true})
	}))

	allowed, err := c.CheckPermission(context.Background(), "u-1", "documents", "read")
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true")
	}
}

// --- Error handling tests ---

func TestAPIError404(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]string{"code": "NOT_FOUND", "message": "user not found"})
	}))

	_, err := c.GetUser(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("expected 404, got status %d", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", apiErr.Code)
	}
}

func TestAPIError429(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"message":"rate limited"}`))
	}))

	_, err := c.Login(context.Background(), &LoginRequest{Username: "x", Password: "x"})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsRateLimited() {
		t.Errorf("expected 429, got status %d", apiErr.StatusCode)
	}
}

func TestAPIError409(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 409, map[string]string{"code": "CONFLICT", "message": "duplicate"})
	}))

	err := c.Logout(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsConflict() {
		t.Errorf("expected 409, got status %d", apiErr.StatusCode)
	}
}

func TestAPIErrorNonJSON(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal server error"))
	}))

	_, err := c.GetUser(context.Background(), "u-1")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Message != "internal server error" {
		t.Errorf("expected raw body as message, got %s", apiErr.Message)
	}
}

// --- API key tests ---

func TestAPIKeyHeader(t *testing.T) {
	var gotKey string
	c := New("http://localhost", WithAPIKey("secret-key"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c.baseURL = srv.URL

	_, _ = c.GetUser(context.Background(), "u-1")
	if gotKey != "secret-key" {
		t.Errorf("expected API key header, got %q", gotKey)
	}
}

// --- Middleware tests ---

func TestMiddlewarePublicPath(t *testing.T) {
	c := New("http://localhost")
	called := false
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}), MiddlewareConfig{PublicPaths: []string{"/healthz"}})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called for public path")
	}
}

func TestMiddlewareMissingAuth(t *testing.T) {
	c := New("http://localhost")
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}), MiddlewareConfig{})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareInvalidScheme(t *testing.T) {
	c := New("http://localhost")
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}), MiddlewareConfig{})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareValidToken(t *testing.T) {
	c := New("http://localhost")
	called := false
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user := UserFromContext(r.Context())
		if user == nil {
			t.Fatal("user should be in context")
		}
		if user.UserID != "user-1" {
			t.Errorf("expected user-1, got %s", user.UserID)
		}
		w.WriteHeader(200)
	}), MiddlewareConfig{TenantID: "t-1"})

	// Build a valid unsigned JWT.
	header := base64URLEncode([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64URLEncode([]byte(`{"sub":"user-1","username":"admin"}`))
	token := header + "." + payload + "."

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMiddlewareTenantHeaderInjection(t *testing.T) {
	c := New("http://localhost")
	var gotTenant string
	handler := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenant = r.Header.Get("X-Tenant-ID")
		w.WriteHeader(200)
	}), MiddlewareConfig{TenantID: "t-1"})

	header := base64URLEncode([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64URLEncode([]byte(`{"sub":"u-1"}`))
	token := header + "." + payload + "."

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotTenant != "t-1" {
		t.Errorf("expected tenant t-1 in header, got %q", gotTenant)
	}
}

// --- RequireRole / RequireScope tests ---

func TestRequireRoleAllowed(t *testing.T) {
	c := New("http://localhost")
	next := c.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	ctx := context.WithValue(context.Background(), ContextKeyUser, &UserInfo{
		UserID: "u-1",
		Roles:  []string{"admin", "editor"},
	})
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireRoleDenied(t *testing.T) {
	c := New("http://localhost")
	next := c.RequireRole("superadmin", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	})

	ctx := context.WithValue(context.Background(), ContextKeyUser, &UserInfo{
		UserID: "u-1",
		Roles:  []string{"editor"},
	})
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireRoleNoUser(t *testing.T) {
	c := New("http://localhost")
	next := c.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireScopeAllowed(t *testing.T) {
	c := New("http://localhost")
	next := c.RequireScope("read", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	ctx := context.WithValue(context.Background(), ContextKeyUser, &UserInfo{
		Scopes: []string{"read", "write"},
	})
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScopeDenied(t *testing.T) {
	c := New("http://localhost")
	next := c.RequireScope("admin", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	})

	ctx := context.WithValue(context.Background(), ContextKeyUser, &UserInfo{
		Scopes: []string{"read"},
	})
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	next.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- Options tests ---

func TestNewClientBaseURL(t *testing.T) {
	c := New("http://localhost:8080/")
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("expected trimmed URL, got %s", c.baseURL)
	}
}

func TestWithAPIKey(t *testing.T) {
	c := New("http://localhost", WithAPIKey("test-key"))
	if c.apiKey != "test-key" {
		t.Errorf("expected test-key, got %s", c.apiKey)
	}
}

func TestWithJWKS(t *testing.T) {
	c := New("http://localhost", WithJWKS(15 * time.Minute))
	if c.jwksURL == "" {
		t.Error("expected jwksURL to be set")
	}
	if c.jwksTTL != 15*time.Minute {
		t.Errorf("expected 15m TTL, got %v", c.jwksTTL)
	}
}

func TestAPIChecks(t *testing.T) {
	e := &APIError{StatusCode: 401}
	if !e.IsUnauthorized() {
		t.Error("expected IsUnauthorized")
	}
	e = &APIError{StatusCode: 403}
	if !e.IsForbidden() {
		t.Error("expected IsForbidden")
	}
}

func TestAPIErrorString(t *testing.T) {
	e := &APIError{StatusCode: 404, Code: "NOT_FOUND", Message: "user not found"}
	s := e.Error()
	if !strings.Contains(s, "NOT_FOUND") {
		t.Errorf("error string should contain code: %s", s)
	}
	if !strings.Contains(s, "404") {
		t.Errorf("error string should contain status: %s", s)
	}
}

func TestAPIErrorStringNoCode(t *testing.T) {
	e := &APIError{StatusCode: 500, Message: "oops"}
	s := e.Error()
	if !strings.Contains(s, "500") {
		t.Errorf("error string should contain status: %s", s)
	}
}

// --- Context cancellation test ---

func TestContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.GetUser(ctx, "u-1")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}


