package social

import (
	"testing"
)

func TestNewMicrosoftConnector(t *testing.T) {
	c := NewMicrosoftConnector("ms-client-id", "ms-client-secret")
	if c.ID() != "microsoft" {
		t.Errorf("expected ID 'microsoft', got %s", c.ID())
	}
	if c.DisplayName() != "Microsoft" {
		t.Errorf("expected DisplayName 'Microsoft', got %s", c.DisplayName())
	}
}

func TestMicrosoftConnector_GetAuthURL(t *testing.T) {
	c := NewMicrosoftConnector("ms-client-id", "ms-client-secret")
	url, err := c.GetAuthURL(nil, "test-state", "https://example.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty auth URL")
	}
	if !containsStr(url, "login.microsoftonline.com") {
		t.Errorf("expected Microsoft login URL, got: %s", url)
	}
	if !containsStr(url, "ms-client-id") {
		t.Error("expected client_id in auth URL")
	}
	if !containsStr(url, "test-state") {
		t.Error("expected state in auth URL")
	}
}

func TestNewGitLabConnector(t *testing.T) {
	c := NewGitLabConnector("gl-client-id", "gl-client-secret", "")
	if c.ID() != "gitlab" {
		t.Errorf("expected ID 'gitlab', got %s", c.ID())
	}
	if c.DisplayName() != "GitLab" {
		t.Errorf("expected DisplayName 'GitLab', got %s", c.DisplayName())
	}
}

func TestGitLabConnector_GetAuthURL(t *testing.T) {
	c := NewGitLabConnector("gl-client-id", "gl-client-secret", "https://gitlab.example.com")
	url, err := c.GetAuthURL(nil, "test-state", "https://example.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty auth URL")
	}
	if !containsStr(url, "gitlab.example.com") {
		t.Errorf("expected self-hosted GitLab URL, got: %s", url)
	}
}

func TestGitLabConnector_DefaultBaseURL(t *testing.T) {
	c := NewGitLabConnector("gl-client-id", "gl-client-secret", "")
	url, err := c.GetAuthURL(nil, "test-state", "https://example.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if !containsStr(url, "gitlab.com") {
		t.Errorf("expected gitlab.com as default, got: %s", url)
	}
}

func TestNewAppleConnector(t *testing.T) {
	c := NewAppleConnector("apple-client-id", "jwt-secret")
	if c.ID() != "apple" {
		t.Errorf("expected ID 'apple', got %s", c.ID())
	}
	if c.DisplayName() != "Apple" {
		t.Errorf("expected DisplayName 'Apple', got %s", c.DisplayName())
	}
}

func TestAppleConnector_GetAuthURL(t *testing.T) {
	c := NewAppleConnector("apple-client-id", "jwt-secret")
	url, err := c.GetAuthURL(nil, "test-state", "https://example.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty auth URL")
	}
	if !containsStr(url, "appleid.apple.com") {
		t.Errorf("expected Apple auth URL, got: %s", url)
	}
	if !containsStr(url, "form_post") {
		t.Error("expected response_mode=form_post for Apple")
	}
}

func TestParseAppleUser(t *testing.T) {
	userJSON := `{"name":{"firstName":"John","lastName":"Doe"},"email":"john@example.com"}`
	name, email := ParseAppleUser(userJSON)
	if name != "John Doe" {
		t.Errorf("expected name 'John Doe', got %s", name)
	}
	if email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %s", email)
	}
}

func TestParseAppleUser_Empty(t *testing.T) {
	name, email := ParseAppleUser("")
	if name != "" || email != "" {
		t.Errorf("expected empty name and email, got %s / %s", name, email)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
