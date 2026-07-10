package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

// === Apple HandleCallback — full mock with id_token ===

func makeMockJWT(payload map[string]any) string {
	header := `{"alg":"RS256","typ":"JWT"}`
	payloadBytes, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(header)) + "." +
		base64.RawURLEncoding.EncodeToString(payloadBytes) + "." +
		"mocksig"
}

func TestAppleHandleCallback_FullMock(t *testing.T) {
	idToken := makeMockJWT(map[string]any{
		"sub":             "apple-123",
		"email":           "user@apple.example.com",
		"email_verified":  "true",
		"is_private_email": "false",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-at",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	conn := NewAppleConnector("test-client", "test-secret")
	ac := conn.(*appleConnector)
	ac.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	info, err := ac.HandleCallback(mockCtx(ts.URL), "code", "state", ts.URL+"/callback")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	if info.ExternalID != "apple-123" {
		t.Errorf("ExternalID = %s, want apple-123", info.ExternalID)
	}
	if info.Email != "user@apple.example.com" {
		t.Errorf("Email = %s", info.Email)
	}
	if info.Provider != "apple" {
		t.Errorf("Provider = %s", info.Provider)
	}
}

func TestAppleHandleCallback_NoIDToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-at",
				"token_type":   "Bearer",
			})
			return
		}
	}))
	defer ts.Close()

	conn := NewAppleConnector("test-client", "test-secret")
	ac := conn.(*appleConnector)
	ac.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	_, err := ac.HandleCallback(mockCtx(ts.URL), "code", "state", ts.URL+"/callback")
	if err == nil {
		t.Error("expected error for missing id_token")
	}
}

func TestAppleHandleCallback_InvalidIDToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-at",
				"token_type":   "Bearer",
				"id_token":     "not.a.valid.jwt.format",
			})
			return
		}
	}))
	defer ts.Close()

	conn := NewAppleConnector("test-client", "test-secret")
	ac := conn.(*appleConnector)
	ac.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	_, err := ac.HandleCallback(mockCtx(ts.URL), "code", "state", ts.URL+"/callback")
	if err == nil {
		t.Error("expected error for invalid id_token")
	}
}

func TestAppleHandleCallback_TokenExchangeError(t *testing.T) {
	conn := NewAppleConnector("test-client", "test-secret")
	ac := conn.(*appleConnector)
	ac.config.Endpoint = oauth2.Endpoint{
		AuthURL:  "http://127.0.0.1:1/auth",
		TokenURL: "http://127.0.0.1:1/token",
	}

	_, err := ac.HandleCallback(context.Background(), "bad-code", "state", "http://localhost/cb")
	if err == nil {
		t.Error("expected error from token exchange failure")
	}
}

// === decodeAppleIDToken edge cases ===

func TestDecodeAppleIDToken_BadFormatV3(t *testing.T) {
	_, err := decodeAppleIDToken("only-two-parts.foo")
	if err == nil {
		t.Error("expected error for invalid JWT format")
	}
}

func TestDecodeAppleIDToken_BadBase64V3(t *testing.T) {
	_, err := decodeAppleIDToken("header.!!!invalidbase64!!!.sig")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodeAppleIDToken_EmptyToken(t *testing.T) {
	_, err := decodeAppleIDToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

// === parseJWTClaims coverage ===

func TestParseJWTClaims_PaddedV3(t *testing.T) {
	// Use standard base64 (with padding) to test the fallback path
	payload, _ := json.Marshal(map[string]any{"sub": "test123", "email": "a@b.c"})
	// Add padding to make it standard base64
	paddedPayload := base64.URLEncoding.EncodeToString(payload)
	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." + paddedPayload + ".sig"

	claims, err := parseJWTClaims(jwt)
	if err != nil {
		t.Fatalf("parseJWTClaims: %v", err)
	}
	if claims["sub"] != "test123" {
		t.Errorf("sub = %v", claims["sub"])
	}
}

func TestParseJWTClaims_BadJSONV3(t *testing.T) {
	jwt := "header.bm90anNvbg.sig" // "notjson" in base64
	_, err := parseJWTClaims(jwt)
	if err == nil {
		t.Error("expected error for invalid JSON in payload")
	}
}

func TestParseJWTClaims_TwoPartsV3(t *testing.T) {
	_, err := parseJWTClaims("only.two")
	if err == nil {
		t.Error("expected error for two-part JWT")
	}
}

func TestSplitJWT_MultiDot(t *testing.T) {
	parts := splitJWT("a.b.c")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
}

// === GitHub fetchPrimaryEmail mock ===

func TestGitHubFetchPrimaryEmail_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{
				{"email": "secondary@example.com", "primary": false},
				{"email": "primary@example.com", "primary": true},
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1, "login": "testuser", "name": "Test User",
				"email": "",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	conn := NewGitHubConnector("test", "test")
	gc := conn.(*githubConnector)
	gc.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	// Use redirectTransport to route /user/emails to mock
	ctx := mockCtx(ts.URL)
	info, err := gc.HandleCallback(ctx, "code", "state", ts.URL+"/callback")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	// Should have found primary email
	if info.Email != "primary@example.com" {
		t.Errorf("Email = %s, want primary@example.com", info.Email)
	}
}

func TestGitHubFetchPrimaryEmail_NoPrimary(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{
				{"email": "first@example.com", "primary": false},
				{"email": "second@example.com", "primary": false},
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1, "login": "testuser", "name": "Test User",
				"email": "",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	conn := NewGitHubConnector("test", "test")
	gc := conn.(*githubConnector)
	gc.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	ctx := mockCtx(ts.URL)
	info, err := gc.HandleCallback(ctx, "code", "state", ts.URL+"/callback")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	// Should fall back to first email
	if info.Email != "first@example.com" {
		t.Errorf("Email = %s, want first@example.com as fallback", info.Email)
	}
}

func TestGitHubFetchPrimaryEmail_EmptyList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1, "login": "testuser", "name": "Test User",
				"email": "",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	conn := NewGitHubConnector("test", "test")
	gc := conn.(*githubConnector)
	gc.config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	ctx := mockCtx(ts.URL)
	info, err := gc.HandleCallback(ctx, "code", "state", ts.URL+"/callback")
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}
	// Empty email list → no email set
	if info.Email != "" {
		t.Errorf("Email = %s, want empty", info.Email)
	}
}

// === ParseAppleUser edge cases ===

func TestParseAppleUser_EmptyJSON(t *testing.T) {
	name, email := ParseAppleUser("")
	if name != "" || email != "" {
		t.Errorf("expected empty name and email")
	}
}

func TestParseAppleUser_BadJSONV3(t *testing.T) {
	name, email := ParseAppleUser("{invalid json}")
	if name != "" || email != "" {
		t.Errorf("expected empty for invalid JSON")
	}
}

func TestParseAppleUser_WithName(t *testing.T) {
	name, email := ParseAppleUser(`{"name":{"firstName":"John","lastName":"Doe"},"email":"john@example.com"}`)
	if name != "John Doe" {
		t.Errorf("name = %s", name)
	}
	if email != "john@example.com" {
		t.Errorf("email = %s", email)
	}
}
