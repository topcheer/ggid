package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantRepository manages tenant persistence.
type TenantRepository struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create inserts a new tenant.
func (r *TenantRepository) Create(ctx context.Context, t *domain.Tenant) error {
	settingsJSON, _ := json.Marshal(t.Settings)
	query := `
		INSERT INTO tenants (name, slug, plan, status, settings, max_users)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		t.Name, t.Slug, t.Plan, t.Status, settingsJSON, t.MaxUsers,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// GetByID retrieves a tenant by ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var settingsBytes []byte
	query := `SELECT id, name, slug, plan, status, settings, max_users, created_at, updated_at FROM tenants WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &settingsBytes, &t.MaxUsers, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "tenant", id.String())
	}
	if len(settingsBytes) > 0 {
		json.Unmarshal(settingsBytes, &t.Settings)
	}
	return t, nil
}

// GetBySlug retrieves a tenant by slug.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var settingsBytes []byte
	query := `SELECT id, name, slug, plan, status, settings, max_users, created_at, updated_at FROM tenants WHERE slug = $1`
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &settingsBytes, &t.MaxUsers, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "tenant", slug)
	}
	if len(settingsBytes) > 0 {
		json.Unmarshal(settingsBytes, &t.Settings)
	}
	return t, nil
}

// Update modifies a tenant's mutable fields.
func (r *TenantRepository) Update(ctx context.Context, t *domain.Tenant) error {
	settingsJSON, _ := json.Marshal(t.Settings)
	query := `
		UPDATE tenants SET name = $2, plan = $3, status = $4, settings = $5, max_users = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	err := r.db.QueryRow(ctx, query, t.ID, t.Name, t.Plan, t.Status, settingsJSON, t.MaxUsers).Scan(&t.UpdatedAt)
	if err != nil {
		return mapErr(err, "tenant", t.ID.String())
	}
	return nil
}

// Delete soft-deletes a tenant.
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE tenants SET status = 'deleted', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	return nil
}
