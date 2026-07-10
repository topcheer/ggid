package social

import (
	"context"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	// Initially empty
	if len(r.List()) != 0 {
		t.Fatalf("expected empty registry, got %v", r.List())
	}

	// Register a mock connector
	mock := &mockConnector{id: "test", name: "Test"}
	r.Register(mock)

	// Verify
	if len(r.List()) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(r.List()))
	}

	got, err := r.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID() != "test" {
		t.Fatalf("expected ID 'test', got '%s'", got.ID())
	}

	// Get non-existent
	_, err = r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent connector")
	}
}

func TestParseJWTClaims(t *testing.T) {
	// A valid JWT: header.payload.signature
	// Payload: {"sub":"123","email":"test@example.com","name":"Test User"}
	// base64url encoded
	payload := `{"sub":"123","email":"test@example.com","name":"Test User"}`
	encoded := base64URLEncode([]byte(payload))
	jwt := "eyJhbGciOiJSUzI1NiJ9." + encoded + ".signature"

	claims, err := parseJWTClaims(jwt)
	if err != nil {
		t.Fatalf("parseJWTClaims failed: %v", err)
	}

	if claims["sub"] != "123" {
		t.Errorf("expected sub='123', got '%v'", claims["sub"])
	}
	if claims["email"] != "test@example.com" {
		t.Errorf("expected email='test@example.com', got '%v'", claims["email"])
	}
	if claims["name"] != "Test User" {
		t.Errorf("expected name='Test User', got '%v'", claims["name"])
	}
}

func TestParseJWTClaims_Invalid(t *testing.T) {
	_, err := parseJWTClaims("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

func TestNewGoogleConnector(t *testing.T) {
	c := NewGoogleConnector("client-id", "client-secret")
	if c.ID() != "google" {
		t.Errorf("expected ID 'google', got '%s'", c.ID())
	}
	if c.DisplayName() != "Google" {
		t.Errorf("expected DisplayName 'Google', got '%s'", c.DisplayName())
	}
}

func TestNewGitHubConnector(t *testing.T) {
	c := NewGitHubConnector("client-id", "client-secret")
	if c.ID() != "github" {
		t.Errorf("expected ID 'github', got '%s'", c.ID())
	}
	if c.DisplayName() != "GitHub" {
		t.Errorf("expected DisplayName 'GitHub', got '%s'", c.DisplayName())
	}
}

func TestNewGenericOIDCConnector(t *testing.T) {
	c := NewGenericOIDCConnector("keycloak", "Keycloak",
		"client-id", "client-secret",
		"https://kc.example.com/auth", "https://kc.example.com/token",
		"https://kc.example.com/userinfo", nil)
	if c.ID() != "keycloak" {
		t.Errorf("expected ID 'keycloak', got '%s'", c.ID())
	}
	if c.DisplayName() != "Keycloak" {
		t.Errorf("expected DisplayName 'Keycloak', got '%s'", c.DisplayName())
	}
}

// --- helpers ---

type mockConnector struct {
	id   string
	name string
}

func (m *mockConnector) ID() string          { return m.id }
func (m *mockConnector) DisplayName() string { return m.name }
func (m *mockConnector) GetAuthURL(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockConnector) HandleCallback(_ context.Context, _, _, _ string) (*UserInfo, error) {
	return nil, nil
}

func base64URLEncode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0)
	for i := 0; i < len(data); i += 3 {
		b1 := data[i]
		result = append(result, alphabet[b1>>2])
		if i+1 < len(data) {
			b2 := data[i+1]
			result = append(result, alphabet[((b1&0x3)<<4)|(b2>>4)])
			if i+2 < len(data) {
				b3 := data[i+2]
				result = append(result, alphabet[((b2&0xF)<<2)|(b3>>6)])
				result = append(result, alphabet[b3&0x3F])
			} else {
				result = append(result, alphabet[(b2&0xF)<<2])
			}
		} else {
			result = append(result, alphabet[(b1&0x3)<<4])
		}
	}
	return string(result)
}
