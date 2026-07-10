package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

// === Google HandleCallback — full mock via redirectTransport ===

func TestV4Google_FullMock_V4(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":            "google-123",
			"email":         "user@gmail.com",
			"name":          "Google User",
			"picture":       "https://lh3.googleusercontent.com/photo.jpg",
			"verified_email": true,
		})
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
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
		t.Errorf("Email: got '%s'", info.Email)
	}
	if info.Name != "Google User" {
		t.Errorf("Name: got '%s'", info.Name)
	}
	if info.AvatarURL == "" {
		t.Error("expected non-empty AvatarURL")
	}
}

func TestV4Google_TokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(context.Background(), "bad-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad token exchange")
	}
}

func TestV4Google_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestV4GoogleHandleCallback_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	}))
	defer ts.Close()

	c := NewGoogleConnector("g-id", "g-secret")
	c.(*googleConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// === Slack HandleCallback — full mock ===

func TestV4Slack_TokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := NewSlackConnector("s-id", "s-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(context.Background(), "bad-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad token exchange")
	}
}

func TestV4Slack_NotOK(t *testing.T) {
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

	c := NewSlackConnector("s-id", "s-secret")
	c.(*slackConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for slack not-ok response")
	}
}

// === LinkedIn HandleCallback — full mock ===

func TestV4LinkedIn_TokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := NewLinkedInConnector("li-id", "li-secret")
	c.(*linkedinConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(context.Background(), "bad-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad token exchange")
	}
}

// === GitLab HandleCallback — full mock ===

func TestV4GitLab_FullMock_V4(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tokenHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         12345,
			"username":   "gitlabuser",
			"email":      "user@gitlab.com",
			"name":       "GitLab User",
			"avatar_url": "https://gitlab.com/avatar.jpg",
		})
	}))
	defer ts.Close()

	c := NewGitLabConnector("gl-id", "gl-secret", ts.URL)
	c.(*gitlabConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	info, err := c.HandleCallback(mockCtx(ts.URL), "mock-code", "st", "http://localhost/cb")
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}
	if info.Provider != "gitlab" {
		t.Errorf("Provider: want 'gitlab', got '%s'", info.Provider)
	}
}

func TestV4GitLab_TokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := NewGitLabConnector("gl-id", "gl-secret", ts.URL)
	c.(*gitlabConnector).config.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/authorize",
		TokenURL: ts.URL + "/token",
	}

	_, err := c.HandleCallback(context.Background(), "bad-code", "st", "http://localhost/cb")
	if err == nil {
		t.Fatal("expected error for bad token exchange")
	}
}

// === parseJWTClaims edge cases ===

func TestParseJWTClaims_MalformedBase64(t *testing.T) {
	_, err := parseJWTClaims("header.!!!.signature")
	if err == nil {
		t.Fatal("expected error for malformed base64")
	}
}

func TestParseJWTClaims_EmptyParts(t *testing.T) {
	_, err := parseJWTClaims("..")
	if err == nil {
		t.Fatal("expected error for empty JWT parts")
	}
}

// suppress unused import
var _ = url.Parse
