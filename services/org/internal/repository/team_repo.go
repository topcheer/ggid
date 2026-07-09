package repository

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TeamRepository manages team persistence.
type TeamRepository struct {
	db *pgxpool.Pool
}

func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create inserts a new team.
func (r *TeamRepository) Create(ctx context.Context, team *domain.Team) error {
	query := `
		INSERT INTO teams (org_id, name, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRow(ctx, query, team.OrgID, team.Name, team.Description, team.CreatedBy).
		Scan(&team.ID, &team.CreatedAt)
}

// GetByID retrieves a team by ID.
func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Team, error) {
	team := &domain.Team{}
	query := `SELECT id, org_id, name, description, created_by, created_at FROM teams WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&team.ID, &team.OrgID, &team.Name, &team.Description, &team.CreatedBy, &team.CreatedAt)
	if err != nil {
		return nil, mapErr(err, "team", id.String())
	}
	return team, nil
}

// ListByOrg lists teams within an organization.
func (r *TeamRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.Team, error) {
	query := `
		SELECT id, org_id, name, description, created_by, created_at
		FROM teams WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	var teams []*domain.Team
	for rows.Next() {
		t := &domain.Team{}
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Description, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

// Delete removes a team.
func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete team: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return notFound("team", id.String())
	}
	return nil
}
