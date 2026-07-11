package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// BackupCode represents a stored (hashed) TOTP backup code.
type BackupCode struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	CodeHash  string
	UsedAt    *time.Time
	CreatedAt time.Time
}

// BackupCodeRepository stores backup codes.
type BackupCodeRepository interface {
	Create(ctx context.Context, codes []*BackupCode) error
	ListUnused(ctx context.Context, tenantID, userID uuid.UUID) ([]*BackupCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteAll(ctx context.Context, tenantID, userID uuid.UUID) error
}

// inMemBackupCodeRepo is the default in-memory implementation.
type inMemBackupCodeRepo struct {
	mu    sync.Mutex
	codes map[uuid.UUID]*BackupCode
}

// NewInMemBackupCodeRepo creates an in-memory backup code repository.
func NewInMemBackupCodeRepo() BackupCodeRepository {
	return &inMemBackupCodeRepo{codes: make(map[uuid.UUID]*BackupCode)}
}

func (r *inMemBackupCodeRepo) Create(_ context.Context, codes []*BackupCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range codes {
		r.codes[c.ID] = c
	}
	return nil
}

func (r *inMemBackupCodeRepo) ListUnused(_ context.Context, _, _ uuid.UUID) ([]*BackupCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*BackupCode
	for _, c := range r.codes {
		if c.UsedAt == nil {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *inMemBackupCodeRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.codes[id]; ok {
		now := time.Now()
		c.UsedAt = &now
		return nil
	}
	return fmt.Errorf("backup code not found")
}

func (r *inMemBackupCodeRepo) DeleteAll(_ context.Context, _, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, c := range r.codes {
		if c.UserID == userID {
			delete(r.codes, id)
		}
	}
	return nil
}

// BackupCodeService manages generation, verification, and invalidation of
// TOTP backup codes. Each user gets 10 single-use codes formatted as
// XXXX-XXXX (8 alphanumeric characters).
type BackupCodeService struct {
	repo BackupCodeRepository
}

// NewBackupCodeService creates a new BackupCodeService.
func NewBackupCodeService(repo BackupCodeRepository) *BackupCodeService {
	return &BackupCodeService{repo: repo}
}

const backupCodeCount = 10

// GenerateBackupCodes creates 10 new backup codes for a user.
// Returns the plaintext codes (shown once to the user) and hashes them for storage.
// Any existing codes are replaced.
func (s *BackupCodeService) GenerateBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required")
	}

	// Delete existing codes.
	_ = s.repo.DeleteAll(ctx, tc.TenantID, userID)

	var plainCodes []string
	var hashed []*BackupCode
	for i := 0; i < backupCodeCount; i++ {
		code := generateBackupCode()
		hash, _ := crypto.HashPassword(code)

		plainCodes = append(plainCodes, code)
		hashed = append(hashed, &BackupCode{
			ID:        uuid.New(),
			TenantID:  tc.TenantID,
			UserID:    userID,
			CodeHash:  hash,
			CreatedAt: time.Now(),
		})
	}

	if err := s.repo.Create(ctx, hashed); err != nil {
		return nil, fmt.Errorf("store backup codes: %w", err)
	}

	return plainCodes, nil
}

// VerifyBackupCode checks a backup code against the user's stored codes.
// If valid, the code is marked as used (single-use enforcement).
func (s *BackupCodeService) VerifyBackupCode(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	codes, err := s.repo.ListUnused(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("list backup codes: %w", err)
	}

	for _, bc := range codes {
		ok, _ := crypto.VerifyPassword(code, bc.CodeHash)
		if ok {
			return s.repo.MarkUsed(ctx, bc.ID)
		}
	}

	return ErrInvalidBackupCode
}

// RemainingBackupCodes returns the count of unused backup codes.
func (s *BackupCodeService) RemainingBackupCodes(ctx context.Context, tenantID, userID uuid.UUID) (int, error) {
	codes, err := s.repo.ListUnused(ctx, tenantID, userID)
	if err != nil {
		return 0, err
	}
	return len(codes), nil
}

// generateBackupCode creates a cryptographically random code in XXXX-XXXX format.
func generateBackupCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand should never fail
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return fmt.Sprintf("%s-%s", string(b[:4]), string(b[4:]))
}

// ErrInvalidBackupCode is returned when a backup code is invalid or already used.
var ErrInvalidBackupCode = fmt.Errorf("invalid or used backup code")

// Suppress unused import guard.
var _ = strings.TrimSpace
