// Package repository provides database-backed implementations for the Identity Service.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// GroupRepo implements domain.GroupRepository using PostgreSQL.
type GroupRepo struct {
	db *sql.DB
}

// NewGroupRepo creates a new GroupRepo.
func NewGroupRepo(db *sql.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

// Ensure GroupRepo satisfies domain.GroupRepository at compile time.
var _ domain.GroupRepository = (*GroupRepo)(nil)

func (r *GroupRepo) CreateGroup(ctx context.Context, group *domain.Group) (*domain.Group, error) {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	query := `
		INSERT INTO scim_groups (id, tenant_id, display_name, external_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query,
		group.ID, group.TenantID, group.DisplayName, nullableString(group.ExternalID),
	).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return group, nil
}

func (r *GroupRepo) GetGroupByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Group, error) {
	g := &domain.Group{}
	query := `SELECT id, tenant_id, display_name, COALESCE(external_id, ''), created_at, updated_at
		FROM scim_groups WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&g.ID, &g.TenantID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	return g, nil
}

func (r *GroupRepo) GetGroupByDisplayName(ctx context.Context, tenantID uuid.UUID, displayName string) (*domain.Group, error) {
	g := &domain.Group{}
	query := `SELECT id, tenant_id, display_name, COALESCE(external_id, ''), created_at, updated_at
		FROM scim_groups WHERE tenant_id = $1 AND display_name = $2 AND deleted_at IS NULL`
	err := r.db.QueryRowContext(ctx, query, tenantID, displayName).Scan(
		&g.ID, &g.TenantID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get group by name: %w", err)
	}
	return g, nil
}

func (r *GroupRepo) UpdateGroup(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateGroupInput) (*domain.Group, error) {
	setParts := []string{"updated_at = NOW()"}
	args := []any{id, tenantID}
	argIdx := 3 //nolint:ineffassign // incremented in conditional branches

	if input.DisplayName != nil {
		setParts = append(setParts, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *input.DisplayName)
		argIdx++
	}
	if input.ExternalID != nil {
		setParts = append(setParts, fmt.Sprintf("external_id = $%d", argIdx))
		args = append(args, *input.ExternalID)
		argIdx++
	}

	query := fmt.Sprintf(`UPDATE scim_groups SET %s WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		RETURNING id, tenant_id, display_name, COALESCE(external_id, ''), created_at, updated_at`,
		strings.Join(setParts, ", "))

	g := &domain.Group{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&g.ID, &g.TenantID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

func (r *GroupRepo) DeleteGroup(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE scim_groups SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

func (r *GroupRepo) ListGroups(ctx context.Context, filter *domain.GroupListFilter) (*domain.GroupListResult, error) {
	where := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{filter.TenantID}
	argIdx := 2

	if filter.Search != "" {
		where = append(where, fmt.Sprintf("display_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scim_groups WHERE %s", strings.Join(where, " AND "))
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count groups: %w", err)
	}

	listQuery := fmt.Sprintf(
		`SELECT id, tenant_id, display_name, COALESCE(external_id, ''), created_at, updated_at
		FROM scim_groups WHERE %s ORDER BY display_name ASC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, filter.PageSize, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []*domain.Group
	for rows.Next() {
		g := &domain.Group{}
		if err := rows.Scan(&g.ID, &g.TenantID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}

	result := &domain.GroupListResult{
		Groups: groups,
		Total:  total,
	}
	if filter.Offset+filter.PageSize < total {
		result.NextOffset = filter.Offset + filter.PageSize
	}
	return result, nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, tenantID, groupID uuid.UUID) ([]*domain.GroupMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.group_id, m.user_id, COALESCE(m.user_type, 'User'),
		 	COALESCE(u.display_name, u.username, ''), m.created_at
		FROM scim_group_members m
		LEFT JOIN users u ON u.id = m.user_id AND u.tenant_id = m.tenant_id
		WHERE m.group_id = $1 AND m.tenant_id = $2 AND m.deleted_at IS NULL`,
		groupID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []*domain.GroupMember
	for rows.Next() {
		m := &domain.GroupMember{}
		if err := rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.UserType, &m.Display, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		m.UserRef = "/scim/v2/Users/" + m.UserID.String()
		members = append(members, m)
	}
	return members, nil
}

func (r *GroupRepo) AddMembers(ctx context.Context, tenantID, groupID uuid.UUID, members []domain.GroupMemberInput) error {
	if len(members) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, m := range members {
		memberType := m.Type
		if memberType == "" {
			memberType = "User"
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO scim_group_members (id, tenant_id, group_id, user_id, user_type, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (tenant_id, group_id, user_id) DO NOTHING`,
			uuid.New(), tenantID, groupID, m.UserID, memberType)
		if err != nil {
			return fmt.Errorf("add member: %w", err)
		}
	}
	return tx.Commit()
}

func (r *GroupRepo) RemoveMembers(ctx context.Context, tenantID, groupID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, uid := range userIDs {
		_, err := tx.ExecContext(ctx,
			`UPDATE scim_group_members SET deleted_at = NOW()
			WHERE group_id = $1 AND tenant_id = $2 AND user_id = $3 AND deleted_at IS NULL`,
			groupID, tenantID, uid)
		if err != nil {
			return fmt.Errorf("remove member: %w", err)
		}
	}
	return tx.Commit()
}

func (r *GroupRepo) GetMemberGroups(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.Group, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT g.id, g.tenant_id, g.display_name, COALESCE(g.external_id, ''), g.created_at, g.updated_at
		FROM scim_groups g
		INNER JOIN scim_group_members m ON m.group_id = g.id AND m.tenant_id = g.tenant_id
		WHERE m.user_id = $1 AND m.tenant_id = $2 AND m.deleted_at IS NULL AND g.deleted_at IS NULL`,
		userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get member groups: %w", err)
	}
	defer rows.Close()

	var groups []*domain.Group
	for rows.Next() {
		g := &domain.Group{}
		if err := rows.Scan(&g.ID, &g.TenantID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// nullableString returns sql.NullString for empty strings.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
