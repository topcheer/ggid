package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AAGUIDRecord represents an approved authenticator entry persisted in PostgreSQL.
type AAGUIDRecord struct {
	AAGUID      string     `json:"aaguid"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"` // approved | denied | deprecated
	AddedBy     string     `json:"added_by"`
	AddedAt     time.Time  `json:"added_at"`
}

// AAGUID status constants.
const (
	AAGUIDStatusApproved   = "approved"
	AAGUIDStatusDenied     = "denied"
	AAGUIDStatusDeprecated = "deprecated"
)

// AAGUIDAllowlistRepository manages AAGUID allowlist entries in PostgreSQL.
type AAGUIDAllowlistRepository struct {
	pool *pgxpool.Pool
}

func NewAAGUIDAllowlistRepository(pool *pgxpool.Pool) *AAGUIDAllowlistRepository {
	return &AAGUIDAllowlistRepository{pool: pool}
}

// EnsureSchema creates the webauthn_aaguid_allowlist table.
func (r *AAGUIDAllowlistRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webauthn_aaguid_allowlist (
			aaguid      TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT,
			status      TEXT NOT NULL DEFAULT 'approved',
			added_by    TEXT,
			added_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

// Add inserts or updates an AAGUID entry.
func (r *AAGUIDAllowlistRepository) Add(ctx context.Context, rec *AAGUIDRecord) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO webauthn_aaguid_allowlist (aaguid, name, description, status, added_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (aaguid) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			added_by = EXCLUDED.added_by`,
		rec.AAGUID, rec.Name, rec.Description, rec.Status, rec.AddedBy,
	)
	return err
}

// Remove deletes an AAGUID from the allowlist.
func (r *AAGUIDAllowlistRepository) Remove(ctx context.Context, aaguid string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM webauthn_aaguid_allowlist WHERE aaguid = $1`, aaguid)
	return err
}

// List returns all allowlist entries.
func (r *AAGUIDAllowlistRepository) List(ctx context.Context) ([]*AAGUIDRecord, error) {
	if r.pool == nil {
		return []*AAGUIDRecord{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT aaguid, name, description, status, added_by, added_at
		FROM webauthn_aaguid_allowlist
		ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*AAGUIDRecord
	for rows.Next() {
		var rec AAGUIDRecord
		if err := rows.Scan(&rec.AAGUID, &rec.Name, &rec.Description, &rec.Status, &rec.AddedBy, &rec.AddedAt); err != nil {
			continue
		}
		records = append(records, &rec)
	}
	return records, nil
}

// IsApproved checks whether an AAGUID is in the approved list.
func (r *AAGUIDAllowlistRepository) IsApproved(ctx context.Context, aaguid string) bool {
	if r.pool == nil || aaguid == "" {
		return true // nil pool or empty AAGUID = allow all (default open).
	}
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status FROM webauthn_aaguid_allowlist WHERE aaguid = $1`, aaguid).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Check if allowlist is empty — empty = allow all.
			var count int
			r.pool.QueryRow(ctx, `SELECT count(*) FROM webauthn_aaguid_allowlist WHERE status = 'approved'`).Scan(&count)
			if count == 0 {
				return true // empty allowlist = allow all.
			}
			return false // non-empty list, AAGUID not found.
		}
		return true // error = fail open.
	}
	return status == AAGUIDStatusApproved
}

// CheckAttestation verifies an AAGUID is approved for registration.
// Returns nil if approved, error with descriptive message if not.
func (r *AAGUIDAllowlistRepository) CheckAttestation(ctx context.Context, aaguid string) error {
	if !r.IsApproved(ctx, aaguid) {
		return ErrAAGUIDNotApproved
	}
	return nil
}

// SeedDefaults populates the allowlist with common FIDO-certified authenticators.
func (r *AAGUIDAllowlistRepository) SeedDefaults(ctx context.Context, addedBy string) error {
	defaults := map[string]struct{ Name, Desc string }{
		"cb69481e-8ff7-4039-93ec-0a2729a154a8": {"YubiKey 5 NFC", "Yubico YubiKey 5 NFC (USB+NFC)"},
		"08987058-cadc-49b2-ab1f-77a3b49c6f9b": {"YubiKey 5C NFC", "Yubico YubiKey 5C NFC"},
		"34f5766d-1536-4a24-9035-52a172e6330d": {"YubiKey 5 Nano", "Yubico YubiKey 5 Nano"},
		"fa2b99dc-9e39-4257-8f92-4a30d23c4df8": {"YubiKey 5C Nano", "Yubico YubiKey 5C Nano"},
		"6028b017-b1d4-4c02-bf5d-a2737c972a47": {"Windows Hello", "Microsoft Windows Hello"},
		"adce0002-35bc-c60a-648b-0b25f1f05503": {"Apple Face ID/Touch ID", "Apple platform authenticator"},
	}
	for aaguid, info := range defaults {
		rec := &AAGUIDRecord{
			AAGUID:      aaguid,
			Name:        info.Name,
			Description: info.Desc,
			Status:      AAGUIDStatusApproved,
			AddedBy:     addedBy,
		}
		if err := r.Add(ctx, rec); err != nil {
			return err
		}
	}
	return nil
}

// GetByID looks up a single AAGUID entry.
func (r *AAGUIDAllowlistRepository) GetByID(ctx context.Context, aaguid string) (*AAGUIDRecord, error) {
	if r.pool == nil {
		return nil, nil
	}
	row := r.pool.QueryRow(ctx, `
		SELECT aaguid, name, description, status, added_by, added_at
		FROM webauthn_aaguid_allowlist WHERE aaguid = $1`, aaguid)
	var rec AAGUIDRecord
	err := row.Scan(&rec.AAGUID, &rec.Name, &rec.Description, &rec.Status, &rec.AddedBy, &rec.AddedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// Unused import guard for uuid (used in handler, not repo, but kept for consistency).
var _ = uuid.New
