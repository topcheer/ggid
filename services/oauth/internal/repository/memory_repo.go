package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrClientNotFound = fmt.Errorf("client not found")
	ErrCodeNotFound   = fmt.Errorf("authorization code not found")
)

// MemoryClientRepository is an in-memory ClientRepository for when DB is unavailable.
type MemoryClientRepository struct {
	mu      sync.RWMutex
	clients map[string]*domain.OAuthClient
}

func NewMemoryClientRepository() *MemoryClientRepository {
	return &MemoryClientRepository{clients: make(map[string]*domain.OAuthClient)}
}

func (r *MemoryClientRepository) CreateClient(_ context.Context, client *domain.OAuthClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client.ClientID] = client
	return nil
}

func (r *MemoryClientRepository) GetClientByID(_ context.Context, _ uuid.UUID, clientID string) (*domain.OAuthClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[clientID]
	if !ok {
		return nil, ErrClientNotFound
	}
	return c, nil
}

func (r *MemoryClientRepository) ListClients(_ context.Context, _ uuid.UUID, pageSize, offset int) ([]*domain.OAuthClient, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*domain.OAuthClient, 0, len(r.clients))
	for _, c := range r.clients {
		all = append(all, c)
	}
	total := len(all)
	if offset >= total {
		return []*domain.OAuthClient{}, total, nil
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	if pageSize <= 0 {
		end = total
	}
	return all[offset:end], total, nil
}

func (r *MemoryClientRepository) UpdateClient(_ context.Context, _ uuid.UUID, clientID string, client *domain.OAuthClient) (*domain.OAuthClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[clientID]; !ok {
		return nil, ErrClientNotFound
	}
	r.clients[clientID] = client
	return client, nil
}

func (r *MemoryClientRepository) DeleteClient(_ context.Context, _ uuid.UUID, clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[clientID]; !ok {
		return ErrClientNotFound
	}
	delete(r.clients, clientID)
	return nil
}

// MemoryCodeRepository is an in-memory AuthorizationCodeRepository.
type MemoryCodeRepository struct {
	mu    sync.RWMutex
	codes map[string]*domain.AuthorizationCode
}

func NewMemoryCodeRepository() *MemoryCodeRepository {
	return &MemoryCodeRepository{codes: make(map[string]*domain.AuthorizationCode)}
}

func (r *MemoryCodeRepository) CreateCode(_ context.Context, code *domain.AuthorizationCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codes[code.CodeHash] = code
	return nil
}

func (r *MemoryCodeRepository) ConsumeCode(_ context.Context, codeHash string) (*domain.AuthorizationCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.codes[codeHash]
	if !ok {
		return nil, ErrCodeNotFound
	}
	delete(r.codes, codeHash)
	return c, nil
}

// MemoryIDTokenRepository is an in-memory IDTokenRepository.
type MemoryIDTokenRepository struct {
	mu       sync.RWMutex
	tokens   map[string]*domain.IDTokenRecord
	refresh  map[string]*domain.RefreshTokenRecord
}

func NewMemoryIDTokenRepository() *MemoryIDTokenRepository {
	return &MemoryIDTokenRepository{
		tokens:  make(map[string]*domain.IDTokenRecord),
		refresh: make(map[string]*domain.RefreshTokenRecord),
	}
}

func (r *MemoryIDTokenRepository) RecordIDToken(_ context.Context, record *domain.IDTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[record.JTI] = record
	return nil
}

func (r *MemoryIDTokenRepository) StoreRefreshToken(_ context.Context, record *domain.RefreshTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := record.TenantID.String() + ":" + record.TokenHash
	r.refresh[key] = record
	return nil
}

func (r *MemoryIDTokenRepository) GetRefreshToken(_ context.Context, tenantID uuid.UUID, tokenHash string) (*domain.RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := tenantID.String() + ":" + tokenHash
	c, ok := r.refresh[key]
	if !ok {
		return nil, fmt.Errorf("refresh token not found")
	}
	return c, nil
}

func (r *MemoryIDTokenRepository) RevokeRefreshToken(_ context.Context, tenantID uuid.UUID, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := tenantID.String() + ":" + tokenHash
	delete(r.refresh, key)
	return nil
}

func (r *MemoryIDTokenRepository) RevokeAllRefreshTokens(_ context.Context, _, _ uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Simplified: clear all (proper impl would filter by clientID)
	r.refresh = make(map[string]*domain.RefreshTokenRecord)
	return nil
}
