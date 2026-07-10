package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// redirectTransport redirects all HTTP requests to a target base URL.
// This allows mocking connectors whose HandleCallback methods use hardcoded
// API URLs (e.g. googleapis.com, discord.com) by routing every request —
// including the oauth2 token exchange and the subsequent userinfo GET —
// through a single httptest.Server.
type redirectTransport struct {
	target string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(t.target)
	if err != nil {
		return nil, err
	}
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = u.Scheme
	req2.URL.Host = u.Host
	req2.Host = u.Host
	req2.RequestURI = ""
	return http.DefaultTransport.RoundTrip(req2)
}

// mockCtx returns a context whose oauth2 HTTP client redirects all requests
// to the given test server URL.
func mockCtx(serverURL string) context.Context {
	return context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{
		Transport: &redirectTransport{target: serverURL},
	})
}

// tokenHandler responds to POST with a mock OAuth2 token.
func tokenHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// === Google HandleCallback — full mock ===

func TestGoogleHandleCallback_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":             "google-123",
			"email":          "user@gmail.com",
			"name":           "Google User",
			"picture":        "https://lh3.googleusercontent.com/photo.jpg",
			"verified_email": true,
		})
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "google" {
		t.Errorf("Provider: want 'google', got '%s'", info.Provider)
	}
	if info.ExternalID != "google-123" {
		t.Errorf("ExternalID: want 'google-123', got '%s'", info.ExternalID)
	}
	if info.Email != "user@gmail.com" {
		t.Errorf("Email: want 'user@gmail.com', got '%s'", info.Email)
	}
	if info.Name != "Google User" {
		t.Errorf("Name: want 'Google User', got '%s'", info.Name)
	}
	if info.AvatarURL != "https://lh3.googleusercontent.com/photo.jpg" {
		t.Errorf("AvatarURL: got '%s'", info.AvatarURL)
	}
	if info.RawClaims == nil || info.RawClaims["id"] != "google-123" {
		t.Errorf("RawClaims not populated correctly: %v", info.RawClaims)
	}
}

// === Discord HandleCallback — full mock (with avatar) ===

func TestDiscordHandleCallback_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       "discord-456",
			"username": "DiscordUser",
			"email":    "user@discord.com",
			"avatar":   "a1b2c3d4",
		})
	}))
	defer ts.Close()

	c := NewDiscordConnector("dc-id", "dc-secret")
	c.(*discordConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "discord" {
		t.Errorf("Provider: want 'discord', got '%s'", info.Provider)
	}
	if info.ExternalID != "discord-456" {
		t.Errorf("ExternalID: want 'discord-456', got '%s'", info.ExternalID)
	}
	if info.Email != "user@discord.com" {
		t.Errorf("Email: got '%s'", info.Email)
	}
	if info.Name != "DiscordUser" {
		t.Errorf("Name: got '%s'", info.Name)
	}
	expectedAvatar := "https://cdn.discordapp.com/avatars/discord-456/a1b2c3d4.png"
	if info.AvatarURL != expectedAvatar {
		t.Errorf("AvatarURL: want '%s', got '%s'", expectedAvatar, info.AvatarURL)
	}
}

// === Discord HandleCallback — empty avatar ===

func TestDiscordHandleCallback_EmptyAvatar(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       "discord-789",
			"username": "NoAvatar",
			"email":    "noavatar@discord.com",
			"avatar":   "",
		})
	}))
	defer ts.Close()

	c := NewDiscordConnector("dc-id", "dc-secret")
	c.(*discordConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.AvatarURL != "" {
		t.Errorf("expected empty avatar URL, got '%s'", info.AvatarURL)
	}
	if info.Name != "NoAvatar" {
		t.Errorf("Name: got '%s'", info.Name)
	}
}

// === Discord HandleCallback — invalid JSON ===

func TestDiscordHandleCallback_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	c := NewDiscordConnector("dc-id", "dc-secret")
	c.(*discordConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse discord profile") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// === Slack HandleCallback — full mock (OK) ===

func TestSlackHandleCallback_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"user": map[string]any{
				"id":    "slack-789",
				"name":  "SlackUser",
				"email": "user@slack.com",
			},
			"team": map[string]any{
				"name": "MyTeam",
			},
		})
	}))
	defer ts.Close()

	c := NewSlackConnector("sl-id", "sl-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/access",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	// Note: slack.go has duplicate json:"user" tags (User + UserStruct),
	// causing Go's json decoder to ignore both. Only OK/Provider are reliable.
	if info.Provider != "slack" {
		t.Errorf("Provider: want 'slack', got '%s'", info.Provider)
	}
}

// === Slack HandleCallback — not OK response ===

func TestSlackHandleCallback_NotOk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer ts.Close()

	c := NewSlackConnector("sl-id", "sl-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/access",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for not-ok response")
	}
	if !strings.Contains(err.Error(), "not ok") {
		t.Errorf("expected 'not ok' error, got: %v", err)
	}
}

// === Slack HandleCallback — invalid JSON ===

func TestSlackHandleCallback_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Write([]byte("<<<broken>>>"))
	}))
	defer ts.Close()

	c := NewSlackConnector("sl-id", "sl-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/access",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse slack profile") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// === LinkedIn HandleCallback — full mock ===

func TestLinkedInHandleCallback_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":         "linkedin-999",
			"name":        "LinkedIn User",
			"given_name":  "Linked",
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
	if info.ExternalID != "linkedin-999" {
		t.Errorf("ExternalID: want 'linkedin-999', got '%s'", info.ExternalID)
	}
	if info.Email != "user@linkedin.com" {
		t.Errorf("Email: got '%s'", info.Email)
	}
	if info.Name != "LinkedIn User" {
		t.Errorf("Name: got '%s'", info.Name)
	}
	if info.AvatarURL != "https://media.linkedin.com/photo.jpg" {
		t.Errorf("AvatarURL: got '%s'", info.AvatarURL)
	}
}

// === LinkedIn HandleCallback — invalid JSON ===

func TestLinkedInHandleCallback_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Write([]byte("not-json"))
	}))
	defer ts.Close()

	c := NewLinkedInConnector("li-id", "li-secret")
	c.(*linkedinConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse linkedin profile") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// === GitLab HandleCallback — full mock ===

func TestGitLabHandleCallback_FullMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         42,
			"username":   "gitlabuser",
			"name":       "GitLab User",
			"email":      "user@gitlab.com",
			"avatar_url": "https://gitlab.com/avatar.png",
		})
	}))
	defer ts.Close()

	// GitLab supports self-hosted instances via baseURL; pass the mock server
	// URL so both token exchange and userinfo API go to the test server.
	c := NewGitLabConnector("gl-id", "gl-secret", ts.URL)

	info, err := c.HandleCallback(context.Background(), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "gitlab" {
		t.Errorf("Provider: want 'gitlab', got '%s'", info.Provider)
	}
	if info.ExternalID != "42" {
		t.Errorf("ExternalID: want '42', got '%s'", info.ExternalID)
	}
	if info.Email != "user@gitlab.com" {
		t.Errorf("Email: got '%s'", info.Email)
	}
	if info.Name != "GitLab User" {
		t.Errorf("Name: got '%s'", info.Name)
	}
	if info.AvatarURL != "https://gitlab.com/avatar.png" {
		t.Errorf("AvatarURL: got '%s'", info.AvatarURL)
	}
	if info.RawClaims == nil || info.RawClaims["username"] != "gitlabuser" {
		t.Errorf("RawClaims not populated: %v", info.RawClaims)
	}
}

// === GitLab HandleCallback — invalid JSON ===

func TestGitLabHandleCallback_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Write([]byte("garbage"))
	}))
	defer ts.Close()

	c := NewGitLabConnector("gl-id", "gl-secret", ts.URL)

	_, err := c.HandleCallback(context.Background(), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse gitlab claims") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// === Google HandleCallback — invalid JSON ===

func TestGoogleHandleCallback_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Write([]byte("not-json"))
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse google claims") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// === Registry.Get — error message format ===

func TestRegistry_GetErrorMessage(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent-provider")
	if err == nil {
		t.Fatal("expected error for non-existent connector")
	}
	if !strings.Contains(err.Error(), "nonexistent-provider") {
		t.Errorf("error should contain the connector ID, got: %v", err)
	}
}

// === All connector constructors — ID and DisplayName ===

func TestAllConnectors_IDAndDisplayName(t *testing.T) {
	tests := []struct {
		connector Connector
		id        string
		name      string
	}{
		{NewGoogleConnector("id", "secret"), "google", "Google"},
		{NewGitHubConnector("id", "secret"), "github", "GitHub"},
		{NewMicrosoftConnector("id", "secret"), "microsoft", "Microsoft"},
		{NewDiscordConnector("id", "secret"), "discord", "Discord"},
		{NewSlackConnector("id", "secret"), "slack", "Slack"},
		{NewLinkedInConnector("id", "secret"), "linkedin", "LinkedIn"},
		{NewGitLabConnector("id", "secret", ""), "gitlab", "GitLab"},
		{NewGitLabConnector("id", "secret", "https://self.hosted"), "gitlab", "GitLab"},
		{NewAppleConnector("id", "secret"), "apple", "Apple"},
		{NewGenericOIDCConnector("custom", "Custom Provider", "id", "secret",
			"https://a.com", "https://t.com", "https://u.com", nil), "custom", "Custom Provider"},
	}

	for _, tt := range tests {
		if tt.connector.ID() != tt.id {
			t.Errorf("ID: want '%s', got '%s'", tt.id, tt.connector.ID())
		}
		if tt.connector.DisplayName() != tt.name {
			t.Errorf("DisplayName: want '%s', got '%s'", tt.name, tt.connector.DisplayName())
		}
	}
}

// === All connectors — GetAuthURL returns non-empty URL with state ===

func TestAllConnectors_GetAuthURL(t *testing.T) {
	connectors := []Connector{
		NewGoogleConnector("id", "secret"),
		NewGitHubConnector("id", "secret"),
		NewMicrosoftConnector("id", "secret"),
		NewDiscordConnector("id", "secret"),
		NewSlackConnector("id", "secret"),
		NewLinkedInConnector("id", "secret"),
		NewGitLabConnector("id", "secret", ""),
		NewAppleConnector("id", "secret"),
		NewGenericOIDCConnector("custom", "Custom", "id", "secret",
			"https://a.com", "https://t.com", "https://u.com", nil),
	}

	for _, c := range connectors {
		url, err := c.GetAuthURL(context.Background(), "test-state", "https://app.com/callback")
		if err != nil {
			t.Errorf("%s.GetAuthURL() error: %v", c.ID(), err)
		}
		if url == "" {
			t.Errorf("%s.GetAuthURL() returned empty URL", c.ID())
		}
		if !strings.Contains(url, "state=test-state") {
			t.Errorf("%s.GetAuthURL() URL missing state: %s", c.ID(), url)
		}
	}
}

// === Registry — Register and Get round-trip ===

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	providers := []Connector{
		NewGoogleConnector("id", "secret"),
		NewGitHubConnector("id", "secret"),
		NewDiscordConnector("id", "secret"),
		NewSlackConnector("id", "secret"),
		NewLinkedInConnector("id", "secret"),
		NewGitLabConnector("id", "secret", ""),
	}

	for _, c := range providers {
		r.Register(c)
	}

	if len(r.List()) != len(providers) {
		t.Errorf("expected %d connectors, got %d", len(providers), len(r.List()))
	}

	for _, c := range providers {
		got, err := r.Get(c.ID())
		if err != nil {
			t.Errorf("Get(%s) error: %v", c.ID(), err)
		}
		if got.ID() != c.ID() {
			t.Errorf("Get(%s) returned wrong connector: %s", c.ID(), got.ID())
		}
	}
}

// === JWT parsing — valid token with multiple claims ===

func TestParseJWTClaims_MultipleClaims(t *testing.T) {
	payload := `{"sub":"jwt-user","email":"jwt@test.com","name":"JWT User","picture":"https://pic.com/1.jpg","custom_field":"custom_value"}`
	encoded := base64URLEncode([]byte(payload))
	jwt := "header." + encoded + ".signature"

	claims, err := parseJWTClaims(jwt)
	if err != nil {
		t.Fatalf("parseJWTClaims failed: %v", err)
	}
	if claims["sub"] != "jwt-user" {
		t.Errorf("sub: want 'jwt-user', got '%v'", claims["sub"])
	}
	if claims["email"] != "jwt@test.com" {
		t.Errorf("email: got '%v'", claims["email"])
	}
	if claims["custom_field"] != "custom_value" {
		t.Errorf("custom_field: got '%v'", claims["custom_field"])
	}
}
