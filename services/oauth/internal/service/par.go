package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// --- RFC 9126: Pushed Authorization Requests (PAR) ---

// PushedAuthorizationRequest holds the parameters for PAR.
type PushedAuthorizationRequest struct {
	TenantID            uuid.UUID
	ClientID            string
	ClientSecret        string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              uuid.UUID
}

// PushedAuthorizationResponse is returned from the PAR endpoint.
type PushedAuthorizationResponse struct {
	RequestURI string `json:"request_uri"` // urn:ietf:params:oauth:request_uri:<uuid>
	ExpiresIn  int    `json:"expires_in"`   // seconds until expiration
}

// parEntry stores a pushed authorization request with its expiry.
type parEntry struct {
	Request *PushedAuthorizationRequest
	ExpiresAt time.Time
}

const (
	parTTL          = 60 // seconds (RFC 9126: SHOULD be short-lived)
	parRequestURIPrefix = "urn:ietf:params:oauth:request_uri:"
)

var (
	parStore sync.Map // requestURI -> parEntry (in-memory fallback)
)

// parStoreRedis stores a PAR entry to Redis with TTL, falling back to in-memory.
func (s *OAuthService) parStoreRedis(ctx context.Context, requestURI string, entry parEntry) {
	parStore.Store(requestURI, entry)
	if s.rdb != nil {
		if data, err := json.Marshal(entry); err == nil {
			s.rdb.Set(ctx, "par:"+requestURI, data, parTTL*time.Second)
		}
	}
}

// parLoadRedis loads a PAR entry from Redis, falling back to in-memory.
func (s *OAuthService) parLoadRedis(ctx context.Context, requestURI string) (parEntry, bool) {
	if val, ok := parStore.Load(requestURI); ok {
		return val.(parEntry), true
	}
	if s.rdb != nil {
		if data, err := s.rdb.Get(ctx, "par:"+requestURI); err == nil && data != "" {
			var entry parEntry
			if json.Unmarshal([]byte(data), &entry) == nil {
				parStore.Store(requestURI, entry)
				return entry, true
			}
		}
	}
	return parEntry{}, false
}

// parDeleteRedis removes a PAR entry from both Redis and in-memory.
func (s *OAuthService) parDeleteRedis(ctx context.Context, requestURI string) {
	parStore.Delete(requestURI)
	if s.rdb != nil {
		s.rdb.Del(ctx, "par:"+requestURI)
	}
}

// PushAuthorizationRequest implements RFC 9126: stores auth params server-side
// and returns a request_uri reference. The /authorize endpoint can then look
// up the stored params using the request_uri.
func (s *OAuthService) PushAuthorizationRequest(ctx context.Context, req *PushedAuthorizationRequest) (*PushedAuthorizationResponse, error) {
	// 1. Validate client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// Verify secret for confidential clients.
	if client.IsConfidential() {
		ok, _ := verifyClientSecret(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 2. Validate redirect URI.
	if !client.ValidateRedirectURI(req.RedirectURI) {
		return nil, errors.InvalidArgument("redirect_uri not registered for client")
	}

	// 3. Validate response type.
	validRT := false
	for _, rt := range client.ResponseTypes {
		if rt == req.ResponseType {
			validRT = true
			break
		}
	}
	if !validRT {
		return nil, errors.InvalidArgument("unsupported response_type")
	}

	// 4. Generate request_uri and store.
	requestURI := parRequestURIPrefix + uuid.New().String()
	s.parStoreRedis(ctx, requestURI, parEntry{
		Request:   req,
		ExpiresAt: time.Now().Add(parTTL * time.Second),
	})

	return &PushedAuthorizationResponse{
		RequestURI: requestURI,
		ExpiresIn:  parTTL,
	}, nil
}

// GetPushedAuthorizationRequest retrieves a pushed authorization request by its request_uri.
// Returns error if not found or expired.
func (s *OAuthService) GetPushedAuthorizationRequest(requestURI string) (*PushedAuthorizationRequest, error) {
	ctx := context.Background()
	if !strings.HasPrefix(requestURI, parRequestURIPrefix) {
		return nil, fmt.Errorf("invalid request_uri format")
	}

	entry, ok := s.parLoadRedis(ctx, requestURI)
	if !ok {
		return nil, fmt.Errorf("request_uri not found or expired")
	}

	if time.Now().After(entry.ExpiresAt) {
		s.parDeleteRedis(ctx, requestURI)
		return nil, fmt.Errorf("request_uri expired")
	}

	// RFC 9126: request_uri is single-use.
	s.parDeleteRedis(ctx, requestURI)
	return entry.Request, nil
}

// verifyClientSecret is a helper that calls crypto.VerifyPassword.
func verifyClientSecret(plaintext, hash string) (bool, error) {
	return crypto.VerifyPassword(plaintext, hash)
}
