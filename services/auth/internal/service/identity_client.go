package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UserInfo represents the minimal user data the Auth Service needs from Identity Service.
type UserInfo struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Username    string
	Email       string
	Status      string // active, locked, disabled, deleted
	DisplayName string
	AvatarURL   string
}

// ExternalIdentityLink represents a linked external identity.
type ExternalIdentityLink struct {
	UserID     uuid.UUID
	Provider   string
	ExternalID string
}

// IdentityClient defines the interface for looking up users from the Identity Service.
// This is a local interface — the real gRPC client will be injected at startup.
type IdentityClient interface {
	// GetUser looks up a user by tenant + username or email.
	GetUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error)
	// GetUserByID looks up a user by ID.
	GetUserByID(ctx context.Context, tenantID, userID uuid.UUID) (*UserInfo, error)
	// FindExternalIdentity finds a user by linked external identity (provider + externalID).
	FindExternalIdentity(ctx context.Context, tenantID uuid.UUID, provider, externalID string) (*ExternalIdentityLink, error)
	// LinkExternalIdentity links a social identity to an existing user.
	LinkExternalIdentity(ctx context.Context, tenantID, userID uuid.UUID, provider, externalID string, metadata map[string]any) error
	// CreateUserFromSocial JIT-provisions a new user from social login.
	CreateUserFromSocial(ctx context.Context, tenantID uuid.UUID, username, email, displayName string, provider, externalID string, metadata map[string]any) (*UserInfo, error)
}

// NoopIdentityClient is a fallback implementation used when the Identity Service
// is not available. It provides degraded-mode operation:
//   - GetUser/GetUserByID: lookups against the local in-memory cache
//   - FindExternalIdentity: checks local identity link cache
//   - LinkExternalIdentity: stores in local cache (best-effort)
//   - CreateUserFromSocial: creates a user record locally (JIT provisioning)
//
// This ensures social login continues to work even when Identity Service is down.
// The local cache is process-scoped and will be lost on restart — when the
// Identity Service comes back, a sync should reconcile these records.
type NoopIdentityClient struct {
	mu          sync.RWMutex
	users       map[uuid.UUID]*UserInfo
	byIdentifier map[string]*UserInfo
	externalLinks map[string]*ExternalIdentityLink
}

// init ensures maps are initialized. Called by all methods so that
// &NoopIdentityClient{} (without NewNoopIdentityClient) is safe to use.
func (n *NoopIdentityClient) init() {
	if n.users == nil {
		n.users = make(map[uuid.UUID]*UserInfo)
		n.byIdentifier = make(map[string]*UserInfo)
		n.externalLinks = make(map[string]*ExternalIdentityLink)
	}
}

// NewNoopIdentityClient creates a new NoopIdentityClient with initialized caches.
func NewNoopIdentityClient() *NoopIdentityClient {
	return &NoopIdentityClient{
		users:        make(map[uuid.UUID]*UserInfo),
		byIdentifier: make(map[string]*UserInfo),
		externalLinks: make(map[string]*ExternalIdentityLink),
	}
}

func (n *NoopIdentityClient) GetUser(_ context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error) {
	n.mu.Lock()
	n.init()
	n.mu.Unlock()
	n.mu.RLock()
	defer n.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", tenantID.String(), identifier)
	if u, ok := n.byIdentifier[key]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %s", identifier)
}

func (n *NoopIdentityClient) GetUserByID(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*UserInfo, error) {
	n.mu.Lock()
	n.init()
	n.mu.Unlock()
	n.mu.RLock()
	defer n.mu.RUnlock()
	if u, ok := n.users[userID]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %s", userID)
}

func (n *NoopIdentityClient) FindExternalIdentity(_ context.Context, tenantID uuid.UUID, provider, externalID string) (*ExternalIdentityLink, error) {
	n.mu.Lock()
	n.init()
	n.mu.Unlock()
	n.mu.RLock()
	defer n.mu.RUnlock()
	key := fmt.Sprintf("%s:%s:%s", tenantID.String(), provider, externalID)
	if link, ok := n.externalLinks[key]; ok {
		return link, nil
	}
	return nil, nil // nil = not found, not an error
}

func (n *NoopIdentityClient) LinkExternalIdentity(_ context.Context, tenantID uuid.UUID, userID uuid.UUID, provider, externalID string, _ map[string]any) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.init()
	key := fmt.Sprintf("%s:%s:%s", tenantID.String(), provider, externalID)
	n.externalLinks[key] = &ExternalIdentityLink{
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
	}
	return nil
}

func (n *NoopIdentityClient) CreateUserFromSocial(_ context.Context, tenantID uuid.UUID, username, email, displayName string, provider, externalID string, _ map[string]any) (*UserInfo, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.init()

	// Check if user already exists by external identity
	extKey := fmt.Sprintf("%s:%s:%s", tenantID.String(), provider, externalID)
	if link, ok := n.externalLinks[extKey]; ok {
		if u, ok := n.users[link.UserID]; ok {
			return u, nil
		}
	}

	// Check if user already exists by email
	idKey := fmt.Sprintf("%s:%s", tenantID.String(), email)
	if u, ok := n.byIdentifier[idKey]; ok {
		// Link external identity to existing user
		n.externalLinks[extKey] = &ExternalIdentityLink{
			UserID:     u.ID,
			Provider:   provider,
			ExternalID: externalID,
		}
		return u, nil
	}

	// Create new user locally (degraded mode)
	userID := uuid.New()
	now := time.Now()
	user := &UserInfo{
		ID:          userID,
		TenantID:    tenantID,
		Username:    username,
		Email:       email,
		Status:      "active",
		DisplayName: displayName,
		AvatarURL:   "",
	}
	_ = now // timestamp not stored in UserInfo currently

	n.users[userID] = user
	n.byIdentifier[fmt.Sprintf("%s:%s", tenantID.String(), username)] = user
	n.byIdentifier[fmt.Sprintf("%s:%s", tenantID.String(), email)] = user
	n.externalLinks[extKey] = &ExternalIdentityLink{
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
	}

	return user, nil
}
