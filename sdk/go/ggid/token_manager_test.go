package ggid

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// --- TokenManager tests ---

// TestTokenManager_NoTokens verifies that AccessToken returns ErrNoTokens
// when Login has not been called.
func TestTokenManager_NoTokens(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	_, err := tm.AccessToken(context.Background())
	if !errors.Is(err, ErrNoTokens) {
		t.Fatalf("expected ErrNoTokens, got %v", err)
	}
}

// TestTokenManager_SetTokens verifies that SetTokens stores tokens correctly.
func TestTokenManager_SetTokens(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	ts := &TokenSet{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    3600,
	}
	tm.SetTokens(ts)

	tokens := tm.GetTokens()
	if tokens == nil || tokens.AccessToken != "test-access-token" {
		t.Fatalf("expected stored token, got %v", tokens)
	}
}

// TestTokenManager_ValidTokenNotRefreshed verifies that a valid token
// (far from expiry) is returned without refresh.
func TestTokenManager_ValidTokenNotRefreshed(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	ts := &TokenSet{
		AccessToken:  "valid-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600, // 1 hour — well above 30s margin
	}
	tm.SetTokens(ts)

	token, err := tm.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "valid-token" {
		t.Fatalf("expected valid-token, got %s", token)
	}
}

// TestTokenManager_NoRefreshToken verifies that when the token is near expiry
// but no refresh token exists, ErrNoRefreshToken is returned.
func TestTokenManager_NoRefreshToken(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	// ExpiresIn=1 means the token is already within the 30s refresh margin
	ts := &TokenSet{
		AccessToken:  "expiring-token",
		RefreshToken: "", // no refresh token
		ExpiresIn:    1,
	}
	tm.SetTokens(ts)

	_, err := tm.AccessToken(context.Background())
	if !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("expected ErrNoRefreshToken, got %v", err)
	}
}

// TestTokenManager_ConcurrentAccess verifies that concurrent calls to
// AccessToken are safe and don't cause panics or data races.
func TestTokenManager_ConcurrentAccess(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	ts := &TokenSet{
		AccessToken:  "concurrent-token",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
	}

	// Set tokens from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.SetTokens(ts)
		}()
	}

	// Read tokens from multiple goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := tm.AccessToken(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if token != "concurrent-token" {
				t.Errorf("expected concurrent-token, got %s", token)
			}
		}()
	}

	wg.Wait()
}

// TestTokenManager_GetTokens verifies that GetTokens returns the stored set.
func TestTokenManager_GetTokens(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	if tm.GetTokens() != nil {
		t.Fatal("expected nil tokens before SetTokens")
	}

	ts := &TokenSet{AccessToken: "abc", ExpiresIn: 100}
	tm.SetTokens(ts)

	got := tm.GetTokens()
	if got == nil || got.AccessToken != "abc" {
		t.Fatalf("expected abc, got %v", got)
	}
}

// TestTokenManager_RefreshMargin verifies the 30s margin logic:
// a token with ExpiresIn > 30s should be returned as-is.
func TestTokenManager_RefreshMargin(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	// Token with 31s left — just above the 30s margin
	ts := &TokenSet{
		AccessToken:  "margin-token",
		RefreshToken: "refresh",
		ExpiresIn:    31,
	}
	tm.SetTokens(ts)

	token, err := tm.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error for 31s token: %v", err)
	}
	if token != "margin-token" {
		t.Fatalf("expected margin-token, got %s", token)
	}
}

// TestTokenManager_RefreshMarginTriggered verifies that a token within the
// margin window (ExpiresIn <= 30) triggers refresh attempt.
func TestTokenManager_RefreshMarginTriggered(t *testing.T) {
	client := NewClient("http://localhost:8080")
	tm := NewTokenManager(client)

	// Token with 29s left — within the 30s margin, no refresh token
	ts := &TokenSet{
		AccessToken:  "about-to-expire",
		RefreshToken: "", // no refresh — should get ErrNoRefreshToken
		ExpiresIn:    29,
	}
	tm.SetTokens(ts)

	_, err := tm.AccessToken(context.Background())
	if !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("expected ErrNoRefreshToken for token within margin, got %v", err)
	}
}

// TestSentinelErrors verifies that sentinel errors implement the error interface
// and can be compared with errors.Is.
func TestSentinelErrors(t *testing.T) {
	if ErrNoTokens.Error() != "ggid: no tokens stored — call Login first" {
		t.Errorf("unexpected ErrNoTokens message: %s", ErrNoTokens.Error())
	}
	if ErrNoRefreshToken.Error() != "ggid: no refresh token available for auto-refresh" {
		t.Errorf("unexpected ErrNoRefreshToken message: %s", ErrNoRefreshToken.Error())
	}

	// Verify errors.Is works
	if !errors.Is(ErrNoTokens, ErrNoTokens) {
		t.Error("errors.Is should return true for same sentinel")
	}
	if !errors.Is(ErrNoRefreshToken, ErrNoRefreshToken) {
		t.Error("errors.Is should return true for same sentinel")
	}
}

// TestRefreshMarginConstant verifies the refresh margin is 30 seconds.
func TestRefreshMarginConstant(t *testing.T) {
	if refreshMargin != 30*time.Second {
		t.Fatalf("expected 30s refresh margin, got %v", refreshMargin)
	}
}
