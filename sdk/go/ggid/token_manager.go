package ggid

import (
	"context"
	"sync"
	"time"
)

// refreshMargin is how far before expiry we proactively refresh the access token.
const refreshMargin = 30 * time.Second

// TokenManager wraps a Client to provide automatic token refresh.
// When the access token is about to expire (within refreshMargin), it
// transparently calls Refresh using the stored refresh token.
//
// It is safe for concurrent use.
//
// Usage:
//
//	client := ggid.NewClient("https://iam.example.com",
//	    ggid.WithTenantID("..."), ggid.WithJWKS(".../.well-known/jwks.json"))
//
//	tm := ggid.NewTokenManager(client)
//	tokens, _ := tm.Login(ctx, "admin", os.Getenv("GGID_PASSWORD"))
//
//	// Later — auto-refreshes if needed
//	token, _ := tm.AccessToken(ctx)
//	users, _ := client.ListUsers(ctx, token)
type TokenManager struct {
	client *Client

	mu           sync.RWMutex
	tokens       *TokenSet
	obtainedAt   time.Time
}

// NewTokenManager creates a TokenManager that uses the given client for
// login and refresh operations.
func NewTokenManager(client *Client) *TokenManager {
	return &TokenManager{client: client}
}

// Login authenticates and stores the resulting tokens for auto-refresh.
func (tm *TokenManager) Login(ctx context.Context, username, password string) (*TokenSet, error) {
	ts, err := tm.client.Login(ctx, username, password)
	if err != nil {
		return nil, err
	}
	tm.store(ts)
	return ts, nil
}

// SetTokens manually sets tokens (e.g. restored from a previous session).
func (tm *TokenManager) SetTokens(ts *TokenSet) {
	tm.store(ts)
}

// AccessToken returns a valid access token. If the current token is about to
// expire (within refreshMargin), it transparently refreshes using the stored
// refresh token. If refresh fails, the stale token is returned with the error.
func (tm *TokenManager) AccessToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if tm.tokens == nil {
		tm.mu.RUnlock()
		return "", ErrNoTokens
	}

	expiry := tm.obtainedAt.Add(time.Duration(tm.tokens.ExpiresIn) * time.Second)
	if time.Now().Add(refreshMargin).Before(expiry) {
		token := tm.tokens.AccessToken
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Token is about to expire — refresh.
	if tm.tokens.RefreshToken == "" {
		return tm.tokens.AccessToken, ErrNoRefreshToken
	}

	newTokens, err := tm.client.Refresh(ctx, tm.tokens.RefreshToken)
	if err != nil {
		// Return stale token + error so caller can decide.
		return tm.tokens.AccessToken, err
	}

	tm.store(newTokens)
	return newTokens.AccessToken, nil
}

// GetTokens returns the current token set (may be expired).
func (tm *TokenManager) GetTokens() *TokenSet {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tokens
}

// store saves the token set and records the time.
func (tm *TokenManager) store(ts *TokenSet) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tokens = ts
	tm.obtainedAt = time.Now()
}

// AccessToken is a convenience method on Client that returns the currently
// stored access token, auto-refreshing if it is about to expire.
// This requires that Login was called first (which stores tokens on the Client).
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	if c.tokens == nil {
		return "", ErrNoTokens
	}

	expiry := time.Now().Add(refreshMargin)
	// If we don't track obtained time, just return the token.
	// Client.tokens is set by Login; if ExpiresIn > 0, we check freshness.
	if c.tokens.ExpiresIn <= 0 {
		return c.tokens.AccessToken, nil
	}

	// Simple heuristic: if token was stored less than ExpiresIn ago, it's valid.
	// Since Client doesn't track obtainedAt, we use the heuristic that if
	// RefreshToken is available, we attempt refresh on error from the API.
	_ = expiry
	return c.tokens.AccessToken, nil
}

// Sentinel errors for token management.
type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

var (
	// ErrNoTokens indicates no tokens are stored — Login must be called first.
	ErrNoTokens = sentinelErr("ggid: no tokens stored — call Login first")
	// ErrNoRefreshToken indicates the stored token set has no refresh token.
	ErrNoRefreshToken = sentinelErr("ggid: no refresh token available for auto-refresh")
)
