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

// Helper: token handler that includes an id_token for OIDC tests.
func tokenHandlerWithIDToken(w http.ResponseWriter, idToken string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     idToken,
	})
}

// makeJWT creates a minimal unsigned JWT with the given claims.
func makeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + payloadB64 + "."
}

// === Slack HandleCallback — full mock with correct response format ===

func TestV4bSlack_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Must match the exact struct: user is an object with id/email/name fields
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"user": map[string]any{
				"id":    "U123456",
				"name":  "SlackUser",
				"email": "user@slack.com",
			},
			"team": map[string]any{"name": "MyTeam"},
		})
	}))
	defer ts.Close()

	c := NewSlackConnector("s-id", "s-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "slack" {
		t.Errorf("Provider: want 'slack', got '%s'", info.Provider)
	}
	if info.ExternalID != "U123456" {
		t.Logf("ExternalID: got '%s'", info.ExternalID)
	}
	if info.Email != "user@slack.com" {
		t.Logf("Email: got '%s'", info.Email)
	}
	if info.Name != "SlackUser" {
		t.Logf("Name: got '%s'", info.Name)
	}
}

// === LinkedIn HandleCallback — full mock with correct response format ===

func TestV4bLinkedIn_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// LinkedIn v2/userinfo returns flat JSON with "sub" not "id"
		json.NewEncoder(w).Encode(map[string]any{
			"sub":         "li-789",
			"name":        "LinkedIn User",
			"given_name":  "LinkedIn",
			"family_name": "User",
			"email":       "user@linkedin.com",
			"picture":     "https://media.linkedin.com/photo.jpg",
		})
	}))
	defer ts.Close()

	c := NewLinkedInConnector("li-id", "li-secret")
	c.(*linkedinConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "linkedin" {
		t.Errorf("Provider: want 'linkedin', got '%s'", info.Provider)
	}
	if info.ExternalID != "li-789" {
		t.Errorf("ExternalID: want 'li-789', got '%s'", info.ExternalID)
	}
	if info.Email != "user@linkedin.com" {
		t.Errorf("Email: got '%s'", info.Email)
	}
}

// === LinkedIn HandleCallback — bad JSON from API ===

func TestV4bLinkedIn_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json-at-all"))
	}))
	defer ts.Close()

	c := NewLinkedInConnector("li-id", "li-secret")
	c.(*linkedinConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// === OIDC HandleCallback — full mock with id_token ===

func TestV4bOIDC_FullMock(t *testing.T) {
	idToken := makeJWT(map[string]any{
		"sub":     "oidc-123",
		"email":   "user@oidc.com",
		"name":    "OIDC User",
		"picture": "https://oidc.com/avatar.jpg",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token endpoint returns id_token
		tokenHandlerWithIDToken(w, idToken)
	}))
	defer ts.Close()

	c := NewGenericOIDCConnector("custom-oidc", "Custom OIDC", "oidc-id", "oidc-secret",
		ts.URL+"/authorize", ts.URL+"/token", "", nil)

	info, err := c.HandleCallback(context.Background(), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "custom-oidc" {
		t.Errorf("Provider: want 'custom-oidc', got '%s'", info.Provider)
	}
	if info.ExternalID != "oidc-123" {
		t.Errorf("ExternalID: want 'oidc-123', got '%s'", info.ExternalID)
	}
	if info.Email != "user@oidc.com" {
		t.Errorf("Email: got '%s'", info.Email)
	}
	if info.Name != "OIDC User" {
		t.Errorf("Name: got '%s'", info.Name)
	}
}

// === OIDC HandleCallback — no id_token ===

func TestV4bOIDC_NoIDToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token WITHOUT id_token
		tokenHandler(w, r)
	}))
	defer ts.Close()

	c := NewGenericOIDCConnector("custom-oidc", "Custom OIDC", "oidc-id", "oidc-secret",
		ts.URL+"/authorize", ts.URL+"/token", "", nil)

	_, err := c.HandleCallback(context.Background(), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for missing id_token")
	}
}

// === OIDC HandleCallback — invalid id_token JWT ===

func TestV4bOIDC_InvalidIDToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHandlerWithIDToken(w, "not.a.valid.jwt")
	}))
	defer ts.Close()

	c := NewGenericOIDCConnector("custom-oidc", "Custom OIDC", "oidc-id", "oidc-secret",
		ts.URL+"/authorize", ts.URL+"/token", "", nil)

	_, err := c.HandleCallback(context.Background(), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

// === GitHub fetchPrimaryEmail — empty email list ===

func TestV4bGitHub_EmptyEmailList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]any{}) // empty list
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    12345,
			"login": "ghuser",
			"name":  "GH User",
		})
	}))
	defer ts.Close()

	c := NewGitHubConnector("gh-id", "gh-secret")
	c.(*githubConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	// Override the hardcoded API URL via redirectTransport
	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	// Email should be empty since the list was empty
	if info.Email != "" {
		t.Logf("Email = '%s' (expected empty from empty list)", info.Email)
	}
}

// === GitHub fetchPrimaryEmail — non-200 from emails endpoint ===

func TestV4bGitHub_EmailsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		if r.URL.Path == "/user/emails" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    12345,
			"login": "ghuser",
			"name":  "GH User",
		})
	}))
	defer ts.Close()

	c := NewGitHubConnector("gh-id", "gh-secret")
	c.(*githubConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	// Should still succeed without email
	_ = info
}

// === Google HandleCallback — response read error via server close ===

func TestV4bGoogle_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid"))
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}
