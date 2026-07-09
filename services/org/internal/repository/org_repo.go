package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrgRepository manages organization tree persistence using LTREE.
type OrgRepository struct {
	db *pgxpool.Pool
}

func NewOrgRepository(db *pgxpool.Pool) *OrgRepository {
	return &OrgRepository{db: db}
}

func ltreeLabel(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "_")
}

// Create inserts a new organization and computes its LTREE path.
func (r *OrgRepository) Create(ctx context.Context, org *domain.Organization) error {
	metaJSON, _ := json.Marshal(org.Metadata)

	var parentPath string
	if org.ParentID != nil {
		err := r.db.QueryRow(ctx, `SELECT path::text FROM organizations WHERE id = $1`, org.ParentID).Scan(&parentPath)
		if err != nil {
			return mapErr(err, "organization (parent)", org.ParentID.String())
		}
	}

	// Insert with temp path, then update with real computed path.
	query := `
		INSERT INTO organizations (tenant_id, parent_id, name, path, metadata)
		VALUES ($1, $2, $3, 'temp'::ltree, $4)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, query, org.TenantID, org.ParentID, org.Name, metaJSON).
		Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	label := ltreeLabel(org.ID)
	if parentPath != "" {
		org.Path = parentPath + "." + label
	} else {
		org.Path = label
	}

	_, err = r.db.Exec(ctx, `UPDATE organizations SET path = $2::ltree WHERE id = $1`, org.ID, org.Path)
	if err != nil {
		return fmt.Errorf("update org path: %w", err)
	}
	return nil
}

// GetByID retrieves an organization by ID.
func (r *OrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	org := &domain.Organization{}
	var metaBytes []byte
	query := `SELECT id, tenant_id, parent_id, name, path::text, metadata, created_at, updated_at FROM organizations WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.TenantID, &org.ParentID, &org.Name, &org.Path, &metaBytes, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "organization", id.String())
	}
	if len(metaBytes) > 0 {
		json.Unmarshal(metaBytes, &org.Metadata)
	}
	return org, nil
}

// ListByTenant lists organizations with pagination.
func (r *OrgRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Organization, error) {
	query := `
		SELECT id, tenant_id, parent_id, name, path::text, metadata, created_at, updated_at
		FROM organizations WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()
	return scanOrgRows(rows)
}

// GetSubTree returns all descendants using LTREE <@ operator.
func (r *OrgRepository) GetSubTree(ctx context.Context, tenantID, rootID uuid.UUID) ([]*domain.Organization, error) {
	var rootPath string
	err := r.db.QueryRow(ctx, `SELECT path::text FROM organizations WHERE id = $1 AND tenant_id = $2`, rootID, tenantID).Scan(&rootPath)
	if err != nil {
		return nil, mapErr(err, "organization", rootID.String())
	}

	query := `
		SELECT id, tenant_id, parent_id, name, path::text, metadata, created_at, updated_at
		FROM organizations
		WHERE tenant_id = $1 AND path <@ $2::ltree
		ORDER BY path`
	rows, err := r.db.Query(ctx, query, tenantID, rootPath)
	if err != nil {
		return nil, fmt.Errorf("get subtree: %w", err)
	}
	defer rows.Close()
	return scanOrgRows(rows)
}

// Update modifies an organization.
func (r *OrgRepository) Update(ctx context.Context, org *domain.Organization) error {
	metaJSON, _ := json.Marshal(org.Metadata)
	query := `UPDATE organizations SET name = $2, metadata = $3, updated_at = NOW() WHERE id = $1 RETURNING updated_at`
	err := r.db.QueryRow(ctx, query, org.ID, org.Name, metaJSON).Scan(&org.UpdatedAt)
	if err != nil {
		return mapErr(err, "organization", org.ID.String())
	}
	return nil
}

// Delete removes an organization (cascade deletes children).
func (r *OrgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return notFound("organization", id.String())
	}
	return nil
}

func scanOrgRows(rows pgx.Rows) ([]*domain.Organization, error) {
	var orgs []*domain.Organization
	for rows.Next() {
		org := &domain.Organization{}
		var metaBytes []byte
		if err := rows.Scan(&org.ID, &org.TenantID, &org.ParentID, &org.Name, &org.Path, &metaBytes, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		if len(metaBytes) > 0 {
			json.Unmarshal(metaBytes, &org.Metadata)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}
