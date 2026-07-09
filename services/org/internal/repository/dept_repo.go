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

// DeptRepository manages department persistence with LTREE hierarchy.
type DeptRepository struct {
	db *pgxpool.Pool
}

func NewDeptRepository(db *pgxpool.Pool) *DeptRepository {
	return &DeptRepository{db: db}
}

// Create inserts a new department and computes its LTREE path.
func (r *DeptRepository) Create(ctx context.Context, dept *domain.Department) error {
	metadataJSON, _ := json.Marshal(dept.Metadata)

	var parentPath string
	if dept.ParentID != nil {
		err := r.db.QueryRow(ctx, `SELECT path::text FROM departments WHERE id = $1`, dept.ParentID).Scan(&parentPath)
		if err != nil {
			return mapErr(err, "department (parent)", dept.ParentID.String())
		}
	}

	query := `
		INSERT INTO departments (org_id, parent_id, name, path, manager_id, metadata)
		VALUES ($1, $2, $3, 'temp'::ltree, $4, $5)
		RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query, dept.OrgID, dept.ParentID, dept.Name, dept.ManagerID, metadataJSON).
		Scan(&dept.ID, &dept.CreatedAt)
	if err != nil {
		return fmt.Errorf("create department: %w", err)
	}

	label := strings.ReplaceAll(dept.ID.String(), "-", "_")
	if parentPath != "" {
		dept.Path = parentPath + "." + label
	} else {
		dept.Path = label
	}

	_, err = r.db.Exec(ctx, `UPDATE departments SET path = $2::ltree WHERE id = $1`, dept.ID, dept.Path)
	if err != nil {
		return fmt.Errorf("update dept path: %w", err)
	}
	return nil
}

// GetByID retrieves a department by ID.
func (r *DeptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	dept := &domain.Department{}
	var metaBytes []byte
	query := `SELECT id, org_id, parent_id, name, path::text, manager_id, metadata, created_at FROM departments WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&dept.ID, &dept.OrgID, &dept.ParentID, &dept.Name, &dept.Path, &dept.ManagerID, &metaBytes, &dept.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "department", id.String())
	}
	if len(metaBytes) > 0 {
		json.Unmarshal(metaBytes, &dept.Metadata)
	}
	return dept, nil
}

// ListByOrg lists departments within an organization.
func (r *DeptRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Department, error) {
	query := `
		SELECT id, org_id, parent_id, name, path::text, manager_id, metadata, created_at
		FROM departments WHERE org_id = $1 ORDER BY path`
	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	defer rows.Close()
	return scanDepts(rows)
}

// Update modifies a department.
func (r *DeptRepository) Update(ctx context.Context, dept *domain.Department) error {
	metadataJSON, _ := json.Marshal(dept.Metadata)
	query := `UPDATE departments SET name = $2, manager_id = $3, metadata = $4 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, dept.ID, dept.Name, dept.ManagerID, metadataJSON)
	if err != nil {
		return mapErr(err, "department", dept.ID.String())
	}
	return nil
}

// Delete removes a department.
func (r *DeptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return notFound("department", id.String())
	}
	return nil
}

func scanDepts(rows pgx.Rows) ([]*domain.Department, error) {
	var depts []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		var metaBytes []byte
		if err := rows.Scan(&d.ID, &d.OrgID, &d.ParentID, &d.Name, &d.Path, &d.ManagerID, &metaBytes, &d.CreatedAt); err != nil {
			return nil, err
		}
		if len(metaBytes) > 0 {
			json.Unmarshal(metaBytes, &d.Metadata)
		}
		depts = append(depts, d)
	}
	return depts, nil
}
