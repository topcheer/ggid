package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

func testClient(id string) *domain.OAuthClient {
	return &domain.OAuthClient{
		ID:       uuid.New(),
		ClientID: id,
		Name:     "Test Client " + id,
		Type:     domain.ClientTypeConfidential,
		Enabled:  true,
	}
}

// --- MemoryClientRepository ---

func TestMemoryClientRepo_CreateAndGet(t *testing.T) {
	repo := NewMemoryClientRepository()
	ctx := context.Background()
	c := testClient("client-1")

	if err := repo.CreateClient(ctx, c); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	got, err := repo.GetClientByID(ctx, uuid.Nil, "client-1")
	if err != nil {
		t.Fatalf("GetClientByID: %v", err)
	}
	if got.ClientID != "client-1" {
		t.Errorf("expected client-1, got %s", got.ClientID)
	}
}

func TestMemoryClientRepo_GetNotFound(t *testing.T) {
	repo := NewMemoryClientRepository()
	_, err := repo.GetClientByID(context.Background(), uuid.Nil, "nonexistent")
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound, got %v", err)
	}
}

func TestMemoryClientRepo_UpdateExisting(t *testing.T) {
	repo := NewMemoryClientRepository()
	ctx := context.Background()
	c := testClient("client-2")
	repo.CreateClient(ctx, c)

	c.Name = "Updated Name"
	updated, err := repo.UpdateClient(ctx, uuid.Nil, "client-2", c)
	if err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Updated Name, got %s", updated.Name)
	}
}

func TestMemoryClientRepo_UpdateNotFound(t *testing.T) {
	repo := NewMemoryClientRepository()
	_, err := repo.UpdateClient(context.Background(), uuid.Nil, "nonexistent", testClient("nonexistent"))
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound for update of missing client, got %v", err)
	}
}

func TestMemoryClientRepo_DeleteExisting(t *testing.T) {
	repo := NewMemoryClientRepository()
	ctx := context.Background()
	repo.CreateClient(ctx, testClient("client-3"))

	if err := repo.DeleteClient(ctx, uuid.Nil, "client-3"); err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}

	// Verify deleted
	_, err := repo.GetClientByID(ctx, uuid.Nil, "client-3")
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound after delete, got %v", err)
	}
}

func TestMemoryClientRepo_DeleteNotFound(t *testing.T) {
	repo := NewMemoryClientRepository()
	err := repo.DeleteClient(context.Background(), uuid.Nil, "nonexistent")
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound for delete of missing client, got %v", err)
	}
}

func TestMemoryClientRepo_ListPagination(t *testing.T) {
	repo := NewMemoryClientRepository()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.CreateClient(ctx, testClient("client-"+string(rune('A'+i))))
	}

	// Page 1: 2 items
	clients, total, err := repo.ListClients(ctx, uuid.Nil, 2, 0)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(clients) != 2 {
		t.Errorf("expected 2 clients on page 1, got %d", len(clients))
	}

	// Page 2: 2 items
	clients2, _, _ := repo.ListClients(ctx, uuid.Nil, 2, 2)
	if len(clients2) != 2 {
		t.Errorf("expected 2 clients on page 2, got %d", len(clients2))
	}

	// Page 3: 1 item
	clients3, _, _ := repo.ListClients(ctx, uuid.Nil, 2, 4)
	if len(clients3) != 1 {
		t.Errorf("expected 1 client on page 3, got %d", len(clients3))
	}

	// Offset beyond total
	clients4, _, _ := repo.ListClients(ctx, uuid.Nil, 2, 10)
	if len(clients4) != 0 {
		t.Errorf("expected 0 clients beyond total, got %d", len(clients4))
	}

	// pageSize=0 returns all
	clients5, _, _ := repo.ListClients(ctx, uuid.Nil, 0, 0)
	if len(clients5) != 5 {
		t.Errorf("expected 5 clients with pageSize=0, got %d", len(clients5))
	}
}

// --- MemoryCodeRepository ---

func TestMemoryCodeRepo_CreateAndConsume(t *testing.T) {
	repo := NewMemoryCodeRepository()
	ctx := context.Background()

	code := &domain.AuthorizationCode{
		ID:        uuid.New(),
		CodeHash:  "hash-123",
		ClientID:  uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := repo.CreateCode(ctx, code); err != nil {
		t.Fatalf("CreateCode: %v", err)
	}

	got, err := repo.ConsumeCode(ctx, "hash-123")
	if err != nil {
		t.Fatalf("ConsumeCode: %v", err)
	}
	if got.CodeHash != "hash-123" {
		t.Errorf("expected hash-123, got %s", got.CodeHash)
	}

	// Consume again should fail (one-time use)
	_, err = repo.ConsumeCode(ctx, "hash-123")
	if err != ErrCodeNotFound {
		t.Errorf("expected ErrCodeNotFound on second consume, got %v", err)
	}
}

func TestMemoryCodeRepo_ConsumeNotFound(t *testing.T) {
	repo := NewMemoryCodeRepository()
	_, err := repo.ConsumeCode(context.Background(), "nonexistent")
	if err != ErrCodeNotFound {
		t.Errorf("expected ErrCodeNotFound, got %v", err)
	}
}

// --- MemoryIDTokenRepository ---

func TestMemoryIDTokenRepo_RecordAndGet(t *testing.T) {
	repo := NewMemoryIDTokenRepository()
	ctx := context.Background()

	record := &domain.IDTokenRecord{
		ID:    uuid.New(),
		JTI:   "jti-abc",
		Scope: []string{"openid", "profile"},
	}
	if err := repo.RecordIDToken(ctx, record); err != nil {
		t.Fatalf("RecordIDToken: %v", err)
	}
}

func TestMemoryIDTokenRepo_RefreshTokenLifecycle(t *testing.T) {
	repo := NewMemoryIDTokenRepository()
	ctx := context.Background()
	tenantID := uuid.New()

	rt := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		TokenHash: "rt-hash-1",
		Scope:     []string{"openid"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Store
	if err := repo.StoreRefreshToken(ctx, rt); err != nil {
		t.Fatalf("StoreRefreshToken: %v", err)
	}

	// Get
	got, err := repo.GetRefreshToken(ctx, tenantID, "rt-hash-1")
	if err != nil {
		t.Fatalf("GetRefreshToken: %v", err)
	}
	if got.TokenHash != "rt-hash-1" {
		t.Errorf("expected rt-hash-1, got %s", got.TokenHash)
	}

	// Revoke single
	if err := repo.RevokeRefreshToken(ctx, tenantID, "rt-hash-1"); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}
	_, err = repo.GetRefreshToken(ctx, tenantID, "rt-hash-1")
	if err == nil {
		t.Error("expected error after revoke, got nil")
	}
}

func TestMemoryIDTokenRepo_GetRefreshNotFound(t *testing.T) {
	repo := NewMemoryIDTokenRepository()
	_, err := repo.GetRefreshToken(context.Background(), uuid.New(), "nonexistent")
	if err == nil {
		t.Error("expected error for missing refresh token, got nil")
	}
}

func TestMemoryIDTokenRepo_RevokeAll(t *testing.T) {
	repo := NewMemoryIDTokenRepository()
	ctx := context.Background()
	tenantID := uuid.New()

	// Store two tokens
	repo.StoreRefreshToken(ctx, &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: tenantID, TokenHash: "rt-1",
	})
	repo.StoreRefreshToken(ctx, &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: tenantID, TokenHash: "rt-2",
	})

	// Revoke all
	if err := repo.RevokeAllRefreshTokens(ctx, tenantID, uuid.New()); err != nil {
		t.Fatalf("RevokeAllRefreshTokens: %v", err)
	}

	// Both should be gone
	_, err1 := repo.GetRefreshToken(ctx, tenantID, "rt-1")
	_, err2 := repo.GetRefreshToken(ctx, tenantID, "rt-2")
	if err1 == nil || err2 == nil {
		t.Error("expected both tokens revoked")
	}
}
