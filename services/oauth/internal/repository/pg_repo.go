// Package repository implements data access for the OAuth Service using pgx/v5.
package repository

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	ggiderrors "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgClientRepo implements ClientRepository.
type pgClientRepo struct {
	pool *pgxpool.Pool
}

// NewPGClientRepository creates a new ClientRepository.
func NewPGClientRepository(pool *pgxpool.Pool) ClientRepository {
	return &pgClientRepo{pool: pool}
}

func setTenantRLS(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String()))
	return err
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return stderrors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isNoRows(err error) bool {
	return stderrors.Is(err, pgx.ErrNoRows)
}

const clientColumns = `
    id, tenant_id, client_id, client_secret_hash, name, type,
    grant_types, response_types, redirect_uris, scopes,
    token_endpoint_auth_method, metadata, enabled, created_at, updated_at`

func scanClient(row pgx.Row) (*domain.OAuthClient, error) {
	c := &domain.OAuthClient{}
	var (
		clientType    string
		grantTypes    []string
		responseTypes []string
		redirectURIs  []string
		scopes        []string
		metadata      []byte
		authMethod    string
	)
	err := row.Scan(
		&c.ID, &c.TenantID, &c.ClientID, &c.ClientSecretHash, &c.Name, &clientType,
		&grantTypes, &responseTypes, &redirectURIs, &scopes,
		&authMethod, &metadata, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Type = domain.ClientType(clientType)
	c.GrantTypes = grantTypes
	c.ResponseTypes = responseTypes
	c.RedirectURIs = redirectURIs
	c.Scopes = scopes
	c.TokenEndpointAuthMethod = authMethod
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &c.Metadata)
	}
	return c, nil
}

// --- Client CRUD ---

const createClientSQL = `
INSERT INTO oauth_clients (id, tenant_id, client_id, client_secret_hash, name, type,
    grant_types, response_types, redirect_uris, scopes,
    token_endpoint_auth_method, metadata, enabled)
VALUES ($1, $2, $3, $4, $5, $6,
    COALESCE($7, '{authorization_code,refresh_token}'::text[]),
    COALESCE($8, '{code}'::text[]),
    COALESCE($9, '{}'::text[]),
    COALESCE($10, '{openid,profile,email}'::text[]),
    $11, $12, $13)
RETURNING created_at, updated_at`

func (r *pgClientRepo) CreateClient(ctx context.Context, client *domain.OAuthClient) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, client.TenantID); err != nil {
		return err
	}

	err = tx.QueryRow(ctx, createClientSQL,
		client.ID, client.TenantID, client.ClientID, client.ClientSecretHash,
		client.Name, string(client.Type),
		client.GrantTypes, client.ResponseTypes, client.RedirectURIs, client.Scopes,
		client.TokenEndpointAuthMethod, client.MetadataJSON(), client.Enabled,
	).Scan(&client.CreatedAt, &client.UpdatedAt)

	if err != nil {
		if isDuplicateKey(err) {
			return ggiderrors.AlreadyExists("client", client.ClientID)
		}
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "create client", err)
	}
	return tx.Commit(ctx)
}

func (r *pgClientRepo) GetClientByID(ctx context.Context, tenantID uuid.UUID, clientID string) (*domain.OAuthClient, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM oauth_clients WHERE client_id = $1`, clientColumns)
	row := tx.QueryRow(ctx, query, clientID)
	client, err := scanClient(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("client", clientID)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "get client", err)
	}

	tx.Commit(ctx)
	return client, nil
}

func (r *pgClientRepo) ListClients(ctx context.Context, tenantID uuid.UUID, pageSize, offset int) ([]*domain.OAuthClient, int, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, 0, err
	}

	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM oauth_clients`).Scan(&total); err != nil {
		return nil, 0, ggiderrors.Wrap(ggiderrors.ErrInternal, "count clients", err)
	}

	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := fmt.Sprintf(`SELECT %s FROM oauth_clients ORDER BY created_at DESC LIMIT $1 OFFSET $2`, clientColumns)
	rows, err := tx.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, ggiderrors.Wrap(ggiderrors.ErrInternal, "list clients", err)
	}
	defer rows.Close()

	var clients []*domain.OAuthClient
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, 0, ggiderrors.Wrap(ggiderrors.ErrInternal, "scan client", err)
		}
		clients = append(clients, c)
	}

	tx.Commit(ctx)
	return clients, total, nil
}

func (r *pgClientRepo) UpdateClient(ctx context.Context, tenantID uuid.UUID, clientID string, client *domain.OAuthClient) (*domain.OAuthClient, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		UPDATE oauth_clients SET
		    name = $3, redirect_uris = $4, scopes = $5, metadata = $6, enabled = $7,
		    updated_at = NOW()
		WHERE client_id = $2 AND tenant_id = $1
		RETURNING %s`, clientColumns)

	row := tx.QueryRow(ctx, query, tenantID, clientID,
		client.Name, client.RedirectURIs, client.Scopes, client.MetadataJSON(), client.Enabled)

	updated, err := scanClient(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("client", clientID)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "update client", err)
	}

	tx.Commit(ctx)
	return updated, nil
}

func (r *pgClientRepo) DeleteClient(ctx context.Context, tenantID uuid.UUID, clientID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	// Support both client_id (gcid_xxx) and internal id (UUID) for deletion
	tag, err := tx.Exec(ctx, `
		DELETE FROM oauth_clients
		WHERE tenant_id = $1 AND (client_id = $2 OR id::text = $2)
	`, tenantID, clientID)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "delete client", err)
	}
	if tag.RowsAffected() == 0 {
		return ggiderrors.NotFound("client", clientID)
	}
	return tx.Commit(ctx)
}

// --- Authorization Code ---

type pgCodeRepo struct {
	pool *pgxpool.Pool
}

// NewPGAuthorizationCodeRepository creates a new AuthorizationCodeRepository.
func NewPGAuthorizationCodeRepository(pool *pgxpool.Pool) AuthorizationCodeRepository {
	return &pgCodeRepo{pool: pool}
}

const createCodeSQL = `
INSERT INTO oauth_authorization_codes
    (id, tenant_id, code_hash, client_id, user_id, redirect_uri, scope,
     code_challenge, code_challenge_method, nonce, expires_at, used)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, false)
RETURNING created_at`

func (r *pgCodeRepo) CreateCode(ctx context.Context, code *domain.AuthorizationCode) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, code.TenantID); err != nil {
		return err
	}

	err = tx.QueryRow(ctx, createCodeSQL,
		code.ID, code.TenantID, code.CodeHash, code.ClientID, code.UserID,
		code.RedirectURI, code.Scope, code.CodeChallenge, code.CodeChallengeMethod,
		code.Nonce, code.ExpiresAt,
	).Scan(&code.CreatedAt)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "create auth code", err)
	}
	return tx.Commit(ctx)
}

const consumeCodeSQL = `
UPDATE oauth_authorization_codes
SET used = true
WHERE code_hash = $1 AND used = false AND expires_at > NOW()
RETURNING id, tenant_id, client_id, user_id, redirect_uri, scope,
          code_challenge, code_challenge_method, nonce, expires_at, created_at`

func (r *pgCodeRepo) ResolveTenantFromCode(ctx context.Context, codeHash string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM oauth_authorization_codes
		WHERE code_hash = $1 AND used = false AND expires_at > NOW()`, codeHash).Scan(&tenantID)
	if err != nil {
		return uuid.Nil, err
	}
	return tenantID, nil
}

func (r *pgCodeRepo) ConsumeCode(ctx context.Context, codeHash string) (*domain.AuthorizationCode, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	// No RLS — code is globally unique via hash.
	code := &domain.AuthorizationCode{}
	err = tx.QueryRow(ctx, consumeCodeSQL, codeHash).
		Scan(&code.ID, &code.TenantID, &code.ClientID, &code.UserID,
			&code.RedirectURI, &code.Scope, &code.CodeChallenge,
			&code.CodeChallengeMethod, &code.Nonce, &code.ExpiresAt, &code.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.New(ggiderrors.ErrInvalidArgument, "invalid or expired authorization code")
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "consume code", err)
	}
	code.CodeHash = codeHash
	return code, tx.Commit(ctx)
}

// --- ID Token Audit ---

type pgIDTokenRepo struct {
	pool *pgxpool.Pool
}

// NewPGIDTokenRepository creates a new IDTokenRepository.
func NewPGIDTokenRepository(pool *pgxpool.Pool) IDTokenRepository {
	return &pgIDTokenRepo{pool: pool}
}

func (r *pgIDTokenRepo) RecordIDToken(ctx context.Context, record *domain.IDTokenRecord) error {
	claimsJSON, _ := json.Marshal(record.Claims)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oidc_id_tokens (id, jti, user_id, client_id, tenant_id, scope, claims, expires_at, issued_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		record.ID, record.JTI, record.UserID, record.ClientID, record.TenantID,
		record.Scope, claimsJSON, record.ExpiresAt, record.IssuedAt)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "record id token", err)
	}
	return nil
}

// Unused import guard
var _ = time.Now

func (r *pgIDTokenRepo) GetRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) (*domain.RefreshTokenRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, client_id, token_hash, scope, expires_at, revoked, used, COALESCE(family_id, ''), created_at
		FROM oidc_refresh_tokens
		WHERE tenant_id = $1 AND token_hash = $2`,
		tenantID, tokenHash)
	var rec domain.RefreshTokenRecord
	var scopeStr string
	err := row.Scan(&rec.ID, &rec.TenantID, &rec.UserID, &rec.ClientID, &rec.TokenHash, &scopeStr, &rec.ExpiresAt, &rec.Revoked, &rec.Used, &rec.FamilyID, &rec.CreatedAt)
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrNotFound, "refresh token not found", err)
	}
	rec.Scope = strings.Fields(scopeStr)
	return &rec, nil
}

func (r *pgIDTokenRepo) RevokeAllRefreshTokens(ctx context.Context, tenantID, clientID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND client_id = $2`, tenantID, clientID)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "revoke all refresh tokens", err)
	}
	return nil
}

func (r *pgIDTokenRepo) StoreRefreshToken(ctx context.Context, record *domain.RefreshTokenRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oidc_refresh_tokens (id, tenant_id, user_id, client_id, token_hash, scope, expires_at, revoked, used, created_at, family_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, false, $8, NULLIF($9, ''))`,
		record.ID, record.TenantID, record.UserID, record.ClientID, record.TokenHash,
		strings.Join(record.Scope, " "), record.ExpiresAt, record.CreatedAt, record.FamilyID)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "store refresh token", err)
	}
	return nil
}

// RevokeRefreshTokensByFamily revokes every refresh token in a rotation
// family (RFC 6749 §10.4 reuse response). It satisfies the service-layer
// FamilyRevoker interface via type assertion.
func (r *pgIDTokenRepo) RevokeRefreshTokensByFamily(ctx context.Context, tenantID uuid.UUID, familyID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND family_id = $2`, tenantID, familyID)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "revoke refresh token family", err)
	}
	return nil
}

func (r *pgIDTokenRepo) RevokeRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE oidc_refresh_tokens SET revoked = true, used = true WHERE tenant_id = $1 AND token_hash = $2`, tenantID, tokenHash)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "revoke refresh token", err)
	}
	return nil
}
