package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// UserInfo represents the minimal user data the Auth Service needs from Identity Service.
type UserInfo struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Username string
	Email    string
	Status   string // active, locked, disabled, deleted
}

// IdentityClient defines the interface for looking up users from the Identity Service.
// This is a local interface — the real gRPC client will be injected at startup.
type IdentityClient interface {
	// GetUser looks up a user by tenant + username or email.
	GetUser(ctx context.Context, tenantID uuid.UUID, identifier string) (*UserInfo, error)
	// GetUserByID looks up a user by ID.
	GetUserByID(ctx context.Context, tenantID, userID uuid.UUID) (*UserInfo, error)
}

// NoopIdentityClient is a stub implementation used when the Identity Service is not available.
// All lookups return an error indicating the service is unreachable.
type NoopIdentityClient struct{}

func (n *NoopIdentityClient) GetUser(_ context.Context, _ uuid.UUID, _ string) (*UserInfo, error) {
	return nil, fmt.Errorf("identity service not configured")
}

func (n *NoopIdentityClient) GetUserByID(_ context.Context, _, _ uuid.UUID) (*UserInfo, error) {
	return nil, fmt.Errorf("identity service not configured")
}
