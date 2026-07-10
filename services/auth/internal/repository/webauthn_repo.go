// Package repository implements the persistence layer for the Auth Service.
package repository

import (
	"context"
	"time"

	"github.com/ggid/ggid/services/auth/internal/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgWebAuthnCredentialStore implements webauthn.CredentialStore using PostgreSQL.
type pgWebAuthnCredentialStore struct {
	pool *pgxpool.Pool
}

// NewWebAuthnCredentialStore creates a DB-backed credential store for WebAuthn.
func NewWebAuthnCredentialStore(pool *pgxpool.Pool) webauthn.CredentialStore {
	return &pgWebAuthnCredentialStore{pool: pool}
}

func (s *pgWebAuthnCredentialStore) SaveCredential(ctx context.Context, cred *webauthn.Credential) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webauthn_credentials (id, tenant_id, user_id, name, credential_id, public_key, transports, counter, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, cred.ID, cred.TenantID, cred.UserID, cred.Name, cred.CredentialID, cred.PublicKey, cred.Transports, cred.Counter, cred.CreatedAt, cred.LastUsedAt)
	return err
}

func (s *pgWebAuthnCredentialStore) GetCredentialsByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*webauthn.Credential, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, user_id, name, credential_id, public_key, transports, counter, created_at, last_used_at
		FROM webauthn_credentials
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*webauthn.Credential
	for rows.Next() {
		c := &webauthn.Credential{}
		var transports []string
		if err := rows.Scan(&c.ID, &c.TenantID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey, &transports, &c.Counter, &c.CreatedAt, &c.LastUsedAt); err != nil {
			return nil, err
		}
		c.Transports = transports
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (s *pgWebAuthnCredentialStore) GetCredentialByID(ctx context.Context, tenantID uuid.UUID, credID []byte) (*webauthn.Credential, error) {
	c := &webauthn.Credential{}
	var transports []string
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, name, credential_id, public_key, transports, counter, created_at, last_used_at
		FROM webauthn_credentials
		WHERE tenant_id = $1 AND credential_id = $2
	`, tenantID, credID).Scan(&c.ID, &c.TenantID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey, &transports, &c.Counter, &c.CreatedAt, &c.LastUsedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Transports = transports
	return c, nil
}

func (s *pgWebAuthnCredentialStore) UpdateCounter(ctx context.Context, tenantID uuid.UUID, credID []byte, counter uint32) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE webauthn_credentials SET counter = $3, last_used_at = $4
		WHERE tenant_id = $1 AND credential_id = $2
	`, tenantID, credID, counter, now)
	return err
}

func (s *pgWebAuthnCredentialStore) UpdateLastUsed(ctx context.Context, tenantID uuid.UUID, credID []byte, lastUsedAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webauthn_credentials SET last_used_at = $3
		WHERE tenant_id = $1 AND credential_id = $2
	`, tenantID, credID, lastUsedAt)
	return err
}

func (s *pgWebAuthnCredentialStore) DeleteCredential(ctx context.Context, tenantID uuid.UUID, credID []byte) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM webauthn_credentials
		WHERE tenant_id = $1 AND credential_id = $2
	`, tenantID, credID)
	return err
}
