package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MembershipRepository manages membership persistence.
type MembershipRepository struct {
	db *pgxpool.Pool
}

func NewMembershipRepository(db *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{db: db}
}

// Create inserts a new membership (invitation).
func (r *MembershipRepository) Create(ctx context.Context, m *domain.Membership) error {
	metaJSON, _ := json.Marshal(m.Metadata)
	query := `
		INSERT INTO memberships (user_id, tenant_id, org_id, dept_id, team_id, title, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	status := m.Status
	if status == "" {
		status = domain.MembershipInvited
	}
	return r.db.QueryRow(ctx, query,
		m.UserID, m.TenantID, m.OrgID, m.DeptID, m.TeamID, m.Title, status, metaJSON,
	).Scan(&m.ID, &m.JoinedAt)
}

// Activate sets a membership status to active and records join time.
func (r *MembershipRepository) Activate(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE memberships SET status = 'active', joined_at = $2 WHERE id = $1 AND status = 'invited'`,
		id, time.Now())
	if err != nil {
		return fmt.Errorf("activate membership: %w", err)
	}
	return nil
}

// Remove sets a membership status to removed.
func (r *MembershipRepository) Remove(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE memberships SET status = 'removed' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("remove membership: %w", err)
	}
	return nil
}

// GetByID retrieves a membership by ID.
func (r *MembershipRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	m := &domain.Membership{}
	var metaBytes []byte
	query := `SELECT id, user_id, tenant_id, org_id, dept_id, team_id, title, status, joined_at, metadata FROM memberships WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.UserID, &m.TenantID, &m.OrgID, &m.DeptID, &m.TeamID, &m.Title, &m.Status, &m.JoinedAt, &metaBytes,
	)
	if err != nil {
		return nil, mapErr(err, "membership", id.String())
	}
	if len(metaBytes) > 0 {
		json.Unmarshal(metaBytes, &m.Metadata)
	}
	return m, nil
}

// ListMembersFilter holds parameters for querying memberships.
type ListMembersFilter struct {
	TenantID uuid.UUID
	OrgID    *uuid.UUID
	DeptID   *uuid.UUID
	TeamID   *uuid.UUID
	Status   domain.MembershipStatus
}

// List returns memberships matching the filter with pagination.
func (r *MembershipRepository) List(ctx context.Context, filter ListMembersFilter, limit, offset int) ([]*domain.Membership, error) {
	query := `SELECT id, user_id, tenant_id, org_id, dept_id, team_id, title, status, joined_at, metadata FROM memberships WHERE tenant_id = $1`
	args := []any{filter.TenantID}
	n := 2

	if filter.OrgID != nil {
		query += fmt.Sprintf(` AND org_id = $%d`, n)
		args = append(args, *filter.OrgID)
		n++
	}
	if filter.DeptID != nil {
		query += fmt.Sprintf(` AND dept_id = $%d`, n)
		args = append(args, *filter.DeptID)
		n++
	}
	if filter.TeamID != nil {
		query += fmt.Sprintf(` AND team_id = $%d`, n)
		args = append(args, *filter.TeamID)
		n++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(` AND status = $%d`, n)
		args = append(args, filter.Status)
		n++
	}

	query += fmt.Sprintf(` ORDER BY joined_at DESC NULLS LAST LIMIT $%d OFFSET $%d`, n, n+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*domain.Membership
	for rows.Next() {
		m := &domain.Membership{}
		var metaBytes []byte
		if err := rows.Scan(&m.ID, &m.UserID, &m.TenantID, &m.OrgID, &m.DeptID, &m.TeamID, &m.Title, &m.Status, &m.JoinedAt, &metaBytes); err != nil {
			return nil, err
		}
		if len(metaBytes) > 0 {
			json.Unmarshal(metaBytes, &m.Metadata)
		}
		memberships = append(memberships, m)
	}
	return memberships, nil
}
