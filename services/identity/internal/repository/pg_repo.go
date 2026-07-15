// Package repository implements data access for the Identity Service using pgx/v5.
package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	ggiderrors "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgRepo implements UserRepository using a pgxpool connection.
type pgRepo struct {
	pool *pgxpool.Pool
}

// NewPGRepository creates a new UserRepository backed by the given pool.
func NewPGRepository(pool *pgxpool.Pool) UserRepository {
	return &pgRepo{pool: pool}
}

// Pool returns the underlying connection pool.
func (r *pgRepo) Pool() *pgxpool.Pool { return r.pool }

// --- Helpers ---

// setTenantRLS sets the app.tenant_id session variable so that PostgreSQL
// Row Level Security policies filter rows automatically.
func setTenantRLS(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String()))
	if err != nil {
		return fmt.Errorf("set tenant RLS: %w", err)
	}
	return nil
}

// isDuplicateKey returns true if err is a PostgreSQL unique violation.
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return stderrors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isNoRows returns true if err is pgx.ErrNoRows.
func isNoRows(err error) bool {
	return stderrors.Is(err, pgx.ErrNoRows)
}

// hashToken returns a hex-encoded SHA-256 hash of the plaintext token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// --- User CRUD ---

const createUserSQL = `
INSERT INTO users (id, tenant_id, username, email, phone, status, email_verified, phone_verified,
    primary_email_id, display_name, avatar_url, locale, timezone, password_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING created_at, updated_at`

func (r *pgRepo) CreateUser(ctx context.Context, user *domain.User) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, user.TenantID); err != nil {
		return err
	}

	var primaryEmailID any
	if user.PrimaryEmailID != nil {
		primaryEmailID = user.PrimaryEmailID
	}

	err = tx.QueryRow(ctx, createUserSQL,
		user.ID, user.TenantID, user.Username, user.Email, user.Phone,
		string(user.Status), user.EmailVerified, user.PhoneVerified,
		primaryEmailID, user.DisplayName, user.AvatarURL, user.Locale,
		user.Timezone, user.PasswordHash,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKey(err) {
			return ggiderrors.AlreadyExists("user", user.Username)
		}
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "create user", err)
	}

	return tx.Commit(ctx)
}

const userColumns = `
    id, tenant_id, username, email, phone, status, email_verified, phone_verified,
    primary_email_id, display_name, avatar_url, locale, timezone,
    last_login_at, last_login_ip, password_hash, created_at, updated_at, deleted_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	var (
		status       string
		primaryEmail *uuid.UUID
		lastLoginAt  *time.Time
		lastLoginIP  *string
		deletedAt    *time.Time
	)

	err := row.Scan(
		&u.ID, &u.TenantID, &u.Username, &u.Email, &u.Phone, &status,
		&u.EmailVerified, &u.PhoneVerified, &primaryEmail, &u.DisplayName,
		&u.AvatarURL, &u.Locale, &u.Timezone, &lastLoginAt, &lastLoginIP,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Status = domain.UserStatus(status)
	u.PrimaryEmailID = primaryEmail
	u.LastLoginAt = lastLoginAt
	u.DeletedAt = deletedAt
	if lastLoginIP != nil {
		if addr, err := netip.ParseAddr(*lastLoginIP); err == nil {
			u.LastLoginIP = &addr
		}
	}
	return u, nil
}

func (r *pgRepo) GetUserByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userColumns), id)
	user, err := scanUser(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("user", id.String())
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "get user", err)
	}

	tx.Rollback(ctx)
	return user, nil
}

func (r *pgRepo) GetUserByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*domain.User, error) {
	return r.getUserByColumn(ctx, tenantID, "username = $1", username)
}

func (r *pgRepo) GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	return r.getUserByColumn(ctx, tenantID, "email = $1", email)
}

func (r *pgRepo) getUserByColumn(ctx context.Context, tenantID uuid.UUID, where, value string) (*domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM users WHERE %s AND deleted_at IS NULL`, userColumns, where)
	row := tx.QueryRow(ctx, query, value)
	user, err := scanUser(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("user", value)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "get user", err)
	}

	tx.Rollback(ctx)
	return user, nil
}

const updateUserSQL = `
UPDATE users SET
    phone = COALESCE($3, phone),
    display_name = COALESCE($4, display_name),
    avatar_url = COALESCE($5, avatar_url),
    locale = COALESCE($6, locale),
    timezone = COALESCE($7, timezone),
    updated_at = NOW()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL
RETURNING %s`

func (r *pgRepo) UpdateUser(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(updateUserSQL, userColumns)
	row := tx.QueryRow(ctx, query, tenantID, id, input.Phone, input.DisplayName,
		input.AvatarURL, input.Locale, input.Timezone)
	user, err := scanUser(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("user", id.String())
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "update user", err)
	}

	tx.Rollback(ctx)
	return user, nil
}

const deleteUserSQL = `
UPDATE users SET status = $3, deleted_at = NOW(), updated_at = NOW()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL`

func (r *pgRepo) DeleteUser(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, deleteUserSQL, tenantID, id, string(domain.UserStatusDeleted))
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "delete user", err)
	}
	if tag.RowsAffected() == 0 {
		return ggiderrors.NotFound("user", id.String())
	}

	return tx.Commit(ctx)
}

func (r *pgRepo) ListUsers(ctx context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, filter.TenantID); err != nil {
		return nil, err
	}

	var (
		where  = []string{"deleted_at IS NULL"}
		args   = []any{}
		argIdx = 1
	)

	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*filter.Status))
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total.
	countQuery := fmt.Sprintf("SELECT count(*) FROM users WHERE %s", whereClause)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "count users", err)
	}

	// Build ORDER BY.
	sortBy := "created_at"
	switch filter.SortBy {
	case "username", "email", "updated_at":
		sortBy = filter.SortBy
	}
	orderDir := "ASC"
	if filter.SortDesc {
		orderDir = "DESC"
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := fmt.Sprintf(
		"SELECT %s FROM users WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		userColumns, whereClause, sortBy, orderDir, argIdx, argIdx+1,
	)
	args = append(args, pageSize, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "list users", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "scan user", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "iterate users", err)
	}

	result := &domain.ListUsersResult{
		Users: users,
		Total: total,
	}
	if filter.Offset+pageSize < total {
		result.NextOffset = filter.Offset + pageSize
	}

	tx.Rollback(ctx)
	return result, nil
}

const setUserStatusSQL = `
UPDATE users SET status = $3, updated_at = NOW()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL
RETURNING %s`

func (r *pgRepo) SetUserStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.UserStatus) (*domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(setUserStatusSQL, userColumns)
	row := tx.QueryRow(ctx, query, tenantID, id, string(status))
	user, err := scanUser(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("user", id.String())
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "set user status", err)
	}

	tx.Rollback(ctx)
	return user, nil
}

const updateLastLoginSQL = `
UPDATE users SET last_login_at = NOW(), last_login_ip = $3, updated_at = NOW()
WHERE id = $2 AND tenant_id = $1`

func (r *pgRepo) UpdateLastLogin(ctx context.Context, tenantID, id uuid.UUID, ip string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, updateLastLoginSQL, tenantID, id, ip)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "update last login", err)
	}

	return tx.Commit(ctx)
}

const updatePasswordSQL = `
UPDATE users SET password_hash = $3, updated_at = NOW()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL`

func (r *pgRepo) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, updatePasswordSQL, tenantID, id, passwordHash)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "update password", err)
	}
	if tag.RowsAffected() == 0 {
		return ggiderrors.NotFound("user", id.String())
	}

	return tx.Commit(ctx)
}

// --- Credential lookup ---

const getCredentialSQL = `
SELECT id, username, email, status, password_hash
FROM users
WHERE tenant_id = $1
  AND (username = $2 OR email = $2)
  AND deleted_at IS NULL`

func (r *pgRepo) GetCredentialByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*authprovider.LocalCredential, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var cred authprovider.LocalCredential
	var status string
	err = tx.QueryRow(ctx, getCredentialSQL, tenantID, username).
		Scan(&cred.UserID, &cred.Username, &cred.Email, &status, &cred.PasswordHash)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("user", username)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "get credential", err)
	}
	cred.Status = status

	tx.Rollback(ctx)
	return &cred, nil
}

// --- Email management ---

const userEmailColumns = `id, user_id, email, is_primary, verified_at, created_at`

func scanUserEmail(row pgx.Row) (*domain.UserEmail, error) {
	e := &domain.UserEmail{}
	err := row.Scan(&e.ID, &e.UserID, &e.Email, &e.IsPrimary, &e.VerifiedAt, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *pgRepo) ListUserEmails(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.UserEmail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM user_emails WHERE user_id = $1 ORDER BY is_primary DESC, created_at ASC`, userEmailColumns)
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "list emails", err)
	}
	defer rows.Close()

	var emails []*domain.UserEmail
	for rows.Next() {
		e, err := scanUserEmail(rows)
		if err != nil {
			return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "scan email", err)
		}
		emails = append(emails, e)
	}

	tx.Rollback(ctx)
	return emails, nil
}

const addUserEmailSQL = `
INSERT INTO user_emails (id, user_id, email, is_primary)
VALUES ($1, $2, $3, false)
RETURNING %s`

func (r *pgRepo) AddUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) (*domain.UserEmail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	emailID := uuid.New()
	query := fmt.Sprintf(addUserEmailSQL, userEmailColumns)
	row := tx.QueryRow(ctx, query, emailID, userID, email)
	ue, err := scanUserEmail(row)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ggiderrors.AlreadyExists("email", email)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "add email", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "commit", err)
	}
	return ue, nil
}

const removeUserEmailSQL = `
DELETE FROM user_emails
WHERE user_id = $2 AND email = $3 AND is_primary = false
  AND user_id IN (SELECT id FROM users WHERE tenant_id = $1)`

func (r *pgRepo) RemoveUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, removeUserEmailSQL, tenantID, userID, email)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "remove email", err)
	}
	if tag.RowsAffected() == 0 {
		return ggiderrors.New(ggiderrors.ErrInvalidArgument, "email not found or cannot remove primary email")
	}

	return tx.Commit(ctx)
}

const setPrimaryEmailSQL = `
UPDATE user_emails SET is_primary = (id = $3)
WHERE user_id = $2
  AND user_id IN (SELECT id FROM users WHERE tenant_id = $1)
RETURNING %s`

func (r *pgRepo) SetPrimaryEmail(ctx context.Context, tenantID, userID, emailID uuid.UUID) (*domain.UserEmail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	// Update users.email to match new primary.
	query := fmt.Sprintf(setPrimaryEmailSQL, userEmailColumns)
	row := tx.QueryRow(ctx, query, tenantID, userID, emailID)
	ue, err := scanUserEmail(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("email", emailID.String())
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "set primary email", err)
	}

	// Also update users.email denormalised field.
	_, err = tx.Exec(ctx, `UPDATE users SET email = $3 WHERE id = $2 AND tenant_id = $1`,
		tenantID, userID, ue.Email)
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "sync user email", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "commit", err)
	}
	return ue, nil
}

func (r *pgRepo) GetUserByEmailID(ctx context.Context, tenantID, emailID uuid.UUID) (*domain.UserEmail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM user_emails WHERE id = $1`, userEmailColumns)
	row := tx.QueryRow(ctx, query, emailID)
	ue, err := scanUserEmail(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("email", emailID.String())
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "get email", err)
	}

	tx.Rollback(ctx)
	return ue, nil
}

// --- Email verification ---

const createVerificationTokenSQL = `
INSERT INTO email_verification_tokens (id, user_id, email_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING created_at`

func (r *pgRepo) CreateEmailVerificationToken(ctx context.Context, token *domain.EmailVerificationToken) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, token.TenantID); err != nil {
		return err
	}

	err = tx.QueryRow(ctx, createVerificationTokenSQL,
		token.ID, token.UserID, token.EmailID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "create verification token", err)
	}

	return tx.Commit(ctx)
}

const consumeVerificationTokenSQL = `
UPDATE email_verification_tokens
SET consumed_at = NOW()
WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > NOW()
RETURNING id, user_id, email_id, expires_at, consumed_at, created_at`

func (r *pgRepo) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	// No RLS needed here — token is globally unique.
	var token domain.EmailVerificationToken
	err = tx.QueryRow(ctx, consumeVerificationTokenSQL, tokenHash).
		Scan(&token.ID, &token.UserID, &token.EmailID, &token.ExpiresAt, &token.ConsumedAt, &token.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.New(ggiderrors.ErrInvalidArgument, "invalid or expired verification token")
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "consume token", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "commit", err)
	}
	return &token, nil
}

// --- External identities ---

const externalIdentityColumns = `id, user_id, provider, external_id, metadata, linked_at`

func scanExternalIdentity(row pgx.Row) (*domain.ExternalIdentity, error) {
	ei := &domain.ExternalIdentity{}
	var metadata []byte
	err := row.Scan(&ei.ID, &ei.UserID, &ei.Provider, &ei.ExternalID, &metadata, &ei.LinkedAt)
	if err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &ei.Metadata)
	}
	return ei, nil
}

func (r *pgRepo) ListExternalIdentities(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.ExternalIdentity, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s FROM user_external_identities WHERE user_id = $1 ORDER BY linked_at DESC`, externalIdentityColumns)
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "list external identities", err)
	}
	defer rows.Close()

	var identities []*domain.ExternalIdentity
	for rows.Next() {
		ei, err := scanExternalIdentity(rows)
		if err != nil {
			return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "scan identity", err)
		}
		identities = append(identities, ei)
	}

	tx.Rollback(ctx)
	return identities, nil
}

const linkExternalIdentitySQL = `
INSERT INTO user_external_identities (id, user_id, provider, external_id, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING %s`

func (r *pgRepo) LinkExternalIdentity(ctx context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, ei.TenantID); err != nil {
		return nil, err
	}

	if ei.ID == uuid.Nil {
		ei.ID = uuid.New()
	}
	query := fmt.Sprintf(linkExternalIdentitySQL, externalIdentityColumns)
	row := tx.QueryRow(ctx, query, ei.ID, ei.UserID, ei.Provider, ei.ExternalID, ei.MetadataJSON())
	result, err := scanExternalIdentity(row)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ggiderrors.AlreadyExists("external identity", ei.Provider+":"+ei.ExternalID)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "link identity", err)
	}
	result.TenantID = ei.TenantID

	if err := tx.Commit(ctx); err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "commit", err)
	}
	return result, nil
}

const unlinkExternalIdentitySQL = `
DELETE FROM user_external_identities
WHERE id = $3 AND user_id = $2
  AND user_id IN (SELECT id FROM users WHERE tenant_id = $1)`

func (r *pgRepo) UnlinkExternalIdentity(ctx context.Context, tenantID, userID, identityID uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, unlinkExternalIdentitySQL, tenantID, userID, identityID)
	if err != nil {
		return ggiderrors.Wrap(ggiderrors.ErrInternal, "unlink identity", err)
	}
	if tag.RowsAffected() == 0 {
		return ggiderrors.NotFound("external identity", identityID.String())
	}

	return tx.Commit(ctx)
}

const findExternalIdentitySQL = `
SELECT %s FROM user_external_identities
WHERE provider = $1 AND external_id = $2
LIMIT 1`

func (r *pgRepo) FindExternalIdentity(ctx context.Context, tenantID uuid.UUID, provider, externalID string) (*domain.ExternalIdentity, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantRLS(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(findExternalIdentitySQL, externalIdentityColumns)
	row := tx.QueryRow(ctx, query, provider, externalID)
	ei, err := scanExternalIdentity(row)
	if err != nil {
		if isNoRows(err) {
			return nil, ggiderrors.NotFound("external identity", provider+":"+externalID)
		}
		return nil, ggiderrors.Wrap(ggiderrors.ErrInternal, "find identity", err)
	}
	ei.TenantID = tenantID

	tx.Rollback(ctx)
	return ei, nil
}
