package tap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TAPRecord represents a Temporary Access Pass.
type TAPRecord struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	CodeHash  string     `json:"-"`          // never expose hash
	IssuedBy  string     `json:"issued_by"`
	Reason    string     `json:"reason"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Engine manages Temporary Access Pass lifecycle.
type Engine struct {
	pool *pgxpool.Pool
}

// NewEngine creates a TAP engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{pool: pool}
}

// EnsureSchema creates temporary_access_passes table.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS temporary_access_passes (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			code_hash TEXT NOT NULL,
			issued_by TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_tap_user ON temporary_access_passes(user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_tap_code_hash ON temporary_access_passes(code_hash) WHERE used_at IS NULL;
	`)
	return err
}

// Issue creates a new TAP for a user.
// Returns the plaintext code (only shown once) and the record.
func (e *Engine) Issue(ctx context.Context, userID, issuedBy, reason string, ttl time.Duration) (string, *TAPRecord, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	code := generateTAPCode()
	codeHash := hashCode(code)

	record := &TAPRecord{
		ID:        fmt.Sprintf("tap-%d", time.Now().UnixNano()),
		UserID:    userID,
		CodeHash:  codeHash,
		IssuedBy:  issuedBy,
		Reason:    reason,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	if e.pool != nil {
		_, err := e.pool.Exec(ctx,
			`INSERT INTO temporary_access_passes (id, user_id, code_hash, issued_by, reason, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			record.ID, record.UserID, record.CodeHash, record.IssuedBy, record.Reason, record.ExpiresAt, record.CreatedAt)
		if err != nil {
			return "", nil, fmt.Errorf("persist TAP: %w", err)
		}
	}

	slog.Info("TAP issued", "user_id", userID, "issued_by", issuedBy, "reason", reason, "expires_at", record.ExpiresAt)
	return code, record, nil
}

// Verify validates a TAP code and marks it as used (single-use enforcement).
// Returns the TAP record if valid.
func (e *Engine) Verify(ctx context.Context, code string) (*TAPRecord, error) {
	codeHash := hashCode(code)

	if e.pool != nil {
		var record TAPRecord
		err := e.pool.QueryRow(ctx,
			`SELECT id, user_id, code_hash, issued_by, reason, expires_at, used_at, created_at
			FROM temporary_access_passes WHERE code_hash = $1 AND used_at IS NULL FOR UPDATE`,
			codeHash).Scan(&record.ID, &record.UserID, &record.CodeHash, &record.IssuedBy, &record.Reason, &record.ExpiresAt, &record.UsedAt, &record.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid or expired TAP")
		}

		// Check expiry.
		if time.Now().After(record.ExpiresAt) {
			return nil, fmt.Errorf("TAP expired")
		}

		// Mark as used.
		now := time.Now()
		if _, err := e.pool.Exec(ctx, `UPDATE temporary_access_passes SET used_at = $1 WHERE id = $2`, now, record.ID); err != nil {
			slog.Error("tap: failed to mark TAP as used", "error", err, "tap_id", record.ID)
		}
		record.UsedAt = &now

		slog.Info("TAP used", "user_id", record.UserID, "tap_id", record.ID)
		return &record, nil
	}

	// Nil pool fallback — no verification possible.
	return nil, fmt.Errorf("no database available for TAP verification")
}

// ListUserTAPs returns TAP history for a user (admin view).
func (e *Engine) ListUserTAPs(ctx context.Context, userID string) ([]TAPRecord, error) {
	if e.pool == nil {
		return nil, nil
	}
	rows, err := e.pool.Query(ctx,
		`SELECT id, user_id, code_hash, issued_by, reason, expires_at, used_at, created_at
		FROM temporary_access_passes WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TAPRecord
	for rows.Next() {
		var r TAPRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.CodeHash, &r.IssuedBy, &r.Reason, &r.ExpiresAt, &r.UsedAt, &r.CreatedAt); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, nil
}

// generateTAPCode creates a random 8-digit code.
func generateTAPCode() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	num := uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	return fmt.Sprintf("%08d", num%100000000)
}

// hashCode returns SHA-256 hex of the code (never store plaintext).
func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
