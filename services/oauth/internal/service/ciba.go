package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// --- OIDC CIBA (Client-Initiated Backchannel Authentication) ---

// BackchannelAuthRequest holds parameters for the CIBA flow.
type BackchannelAuthRequest struct {
	TenantID            uuid.UUID
	ClientID            string
	ClientSecret        string
	Scope               string
	ACRValues           string
	LoginHint           string // hint: username, email, phone number
	LoginHintToken      string // hint: JWT or opaque token
	IDTokenHint         string // hint: existing ID token
	BindingMessage      string // user-friendly message to display on auth device
	UserCode            string // PIN the user must enter
	RequestedExpiry     int    // requested lifetime of auth_req_id in seconds
	Context             string // opaque context for the consumption device
}

// BackchannelAuthResponse is returned from the CIBA backchannel authentication endpoint.
type BackchannelAuthResponse struct {
	AuthReqID string `json:"auth_req_id"` // identifier to poll the token endpoint
	ExpiresIn int    `json:"expires_in"`   // seconds until auth_req_id expires
	Interval  int    `json:"interval"`    // minimum polling interval in seconds
}

// CIBAStatus represents the status of a CIBA authentication request.
type CIBAStatus string

const (
	CIBAStatusPending    CIBAStatus = "pending"
	CIBAStatusApproved   CIBAStatus = "approved"
	CIBAStatusDenied     CIBAStatus = "denied"
	CIBAStatusExpired    CIBAStatus = "expired"
)

// cibaEntry stores a CIBA authentication request.
type cibaEntry struct {
	ClientID       uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Status         CIBAStatus
	BindingMessage string
	Scope          string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastPoll       time.Time
}

const (
	cibaDefaultExpiry = 300 // 5 minutes
	cibaDefaultInterval = 5 // 5 seconds minimum polling
)

var (
	cibaStore sync.Map // authReqID -> cibaEntry (in-memory fallback)
)

// cibaStoreRedis stores a CIBA entry to Redis with TTL, falling back to in-memory.
func (s *OAuthService) cibaStoreRedis(ctx context.Context, authReqID string, entry cibaEntry, ttl time.Duration) {
	cibaStore.Store(authReqID, entry)
	if s.rdb != nil {
		if data, err := json.Marshal(entry); err == nil {
			s.rdb.Set(ctx, "ciba:session:"+authReqID, data, ttl)
		}
	}
}

// cibaLoadRedis loads a CIBA entry from Redis, falling back to in-memory.
func (s *OAuthService) cibaLoadRedis(ctx context.Context, authReqID string) (cibaEntry, bool) {
	if val, ok := cibaStore.Load(authReqID); ok {
		return val.(cibaEntry), true
	}
	if s.rdb != nil {
		if data, err := s.rdb.Get(ctx, "ciba:session:"+authReqID); err == nil && data != "" {
			var entry cibaEntry
			if json.Unmarshal([]byte(data), &entry) == nil {
				cibaStore.Store(authReqID, entry)
				return entry, true
			}
		}
	}
	return cibaEntry{}, false
}

// cibaDeleteRedis removes a CIBA entry from both Redis and in-memory.
func (s *OAuthService) cibaDeleteRedis(ctx context.Context, authReqID string) {
	cibaStore.Delete(authReqID)
	if s.rdb != nil {
		s.rdb.Del(ctx, "ciba:session:"+authReqID)
	}
}

// BackchannelAuthentication implements the OIDC CIBA flow: accepts
// authentication parameters, returns auth_req_id for polling.
func (s *OAuthService) BackchannelAuthentication(ctx context.Context, req *BackchannelAuthRequest) (*BackchannelAuthResponse, error) {
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

	// 2. Check CIBA grant type support.
	if !client.SupportsGrantType("urn:openid:params:grant-type:ciba") {
		return nil, errors.InvalidArgument("client does not support CIBA flow")
	}

	// 3. Require at least one user identification hint.
	if req.LoginHint == "" && req.LoginHintToken == "" && req.IDTokenHint == "" {
		return nil, errors.InvalidArgument("at least one of login_hint, login_hint_token, id_token_hint is required")
	}

	// 4. Determine expiry.
	expiry := cibaDefaultExpiry
	if req.RequestedExpiry > 0 && req.RequestedExpiry <= 900 {
		expiry = req.RequestedExpiry
	}

	// 5. Generate auth_req_id.
	authReqID := generateAuthReqID()

	// 6. Resolve user from hint (resolve userID from login_hint).
	var userID uuid.UUID
	if req.LoginHint != "" {
		// Try to parse as UUID, or derive a synthetic ID.
		if u, err := uuid.Parse(req.LoginHint); err == nil {
			userID = u
		} else {
			userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("ciba:"+req.LoginHint))
		}
	} else {
		userID = uuid.New()
	}

	s.cibaStoreRedis(ctx, authReqID, cibaEntry{
		ClientID:       client.ID,
		TenantID:       req.TenantID,
		UserID:         userID,
		Status:         CIBAStatusPending,
		BindingMessage: req.BindingMessage,
		Scope:          req.Scope,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(time.Duration(expiry) * time.Second),
	}, time.Duration(expiry)*time.Second)

	return &BackchannelAuthResponse{
		AuthReqID: authReqID,
		ExpiresIn: expiry,
		Interval:  cibaDefaultInterval,
	}, nil
}

// PollCIBAToken polls for completion of a CIBA authentication request.
// Returns a token response if the user approved, or an error indicating
// pending, slow_down, or expired.
func (s *OAuthService) PollCIBAToken(ctx context.Context, tenantID uuid.UUID, authReqID, clientID, clientSecret string) (*TokenResponse, error) {
	entry, ok := s.cibaLoadRedis(ctx, authReqID)
	if !ok {
		return nil, &CIBAError{Err: "invalid_grant", Desc: "unknown or expired auth_req_id"}
	}

	// Check expiry.
	if time.Now().After(entry.ExpiresAt) {
		s.cibaDeleteRedis(ctx, authReqID)
		return nil, &CIBAError{Err: "expired_token", Desc: "auth_req_id has expired"}
	}

	// Check polling interval (slow_down).
	if !entry.LastPoll.IsZero() && time.Since(entry.LastPoll) < time.Duration(cibaDefaultInterval)*time.Second {
		return nil, &CIBAError{Err: "slow_down", Desc: "polling too fast"}
	}

	// Update last poll time.
	entry.LastPoll = time.Now()
	s.cibaStoreRedis(ctx, authReqID, entry, time.Until(entry.ExpiresAt))

	// Check status.
	switch entry.Status {
	case CIBAStatusPending:
		return nil, &CIBAError{Err: "authorization_pending", Desc: "user has not yet responded"}

	case CIBAStatusApproved:
		s.cibaDeleteRedis(ctx, authReqID)
		// Issue access token for the resolved user.
		scopes := strings.Fields(entry.Scope)
		accessToken, expiresIn, err := s.issueAccessToken(entry.UserID, entry.TenantID, clientID, entry.Scope)
		if err != nil {
			return nil, errors.Internal("issue CIBA access token", err)
		}
		return &TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			Scope:       joinScopes(scopes),
		}, nil

	case CIBAStatusDenied:
		s.cibaDeleteRedis(ctx, authReqID)
		return nil, &CIBAError{Err: "access_denied", Desc: "user denied the authentication request"}

	default:
		return nil, &CIBAError{Err: "invalid_grant", Desc: "unexpected CIBA status"}
	}
}

// ApproveCIBAAuth marks a CIBA authentication request as approved.
// This is called by the authentication device (e.g., mobile app) after user consent.
func (s *OAuthService) ApproveCIBAAuth(authReqID string) error {
	ctx := context.Background()
	entry, ok := s.cibaLoadRedis(ctx, authReqID)
	if !ok {
		return fmt.Errorf("auth_req_id not found")
	}
	if time.Now().After(entry.ExpiresAt) {
		s.cibaDeleteRedis(ctx, authReqID)
		return fmt.Errorf("auth_req_id expired")
	}
	entry.Status = CIBAStatusApproved
	s.cibaStoreRedis(ctx, authReqID, entry, time.Until(entry.ExpiresAt))
	return nil
}

// DenyCIBAAuth marks a CIBA authentication request as denied.
func (s *OAuthService) DenyCIBAAuth(authReqID string) error {
	ctx := context.Background()
	entry, ok := s.cibaLoadRedis(ctx, authReqID)
	if !ok {
		return fmt.Errorf("auth_req_id not found")
	}
	entry.Status = CIBAStatusDenied
	s.cibaStoreRedis(ctx, authReqID, entry, time.Until(entry.ExpiresAt))
	return nil
}

// CIBAError implements the OAuth2 error response for CIBA polling.
type CIBAError struct {
	Err  string `json:"error"`
	Desc string `json:"error_description"`
}

func (e *CIBAError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.Desc)
}

func generateAuthReqID() string {
	return uuid.New().String() + "-" + generateRandomString(32)
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charsetForToken[cryptoRandInt(len(charsetForToken))]
	}
	return string(b)
}

var charsetForToken = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
