package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportJob represents an async user import job.
type ImportJob struct {
	ID          string             `json:"id"`
	TenantID    uuid.UUID          `json:"tenant_id"`
	Format      string             `json:"format"` // json or csv
	Status      string             `json:"status"` // pending, processing, completed, failed
	Total       int                `json:"total"`
	Imported    int                `json:"imported"`
	Failed      int                `json:"failed"`
	Errors      []ImportRowError   `json:"errors,omitempty"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ImportRowError records a failed row during import.
type ImportRowError struct {
	Row     int    `json:"row"`
	Username string `json:"username,omitempty"`
	Error   string `json:"error"`
}

// importJobRepo manages import_jobs in PostgreSQL.
type importJobRepo struct {
	pool *pgxpool.Pool
}

func newImportJobRepo(pool *pgxpool.Pool) *importJobRepo {
	return &importJobRepo{pool: pool}
}

func (r *importJobRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS import_jobs (
			id           TEXT PRIMARY KEY,
			tenant_id    UUID NOT NULL,
			format       TEXT NOT NULL DEFAULT 'json',
			status       TEXT NOT NULL DEFAULT 'pending',
			total        INTEGER NOT NULL DEFAULT 0,
			imported     INTEGER NOT NULL DEFAULT 0,
			failed       INTEGER NOT NULL DEFAULT 0,
			errors       JSONB DEFAULT '[]'::jsonb,
			started_at   TIMESTAMPTZ,
			completed_at TIMESTAMPTZ,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant ON import_jobs(tenant_id);
	`)
	return err
}

func (r *importJobRepo) Create(ctx context.Context, job *ImportJob) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO import_jobs (id, tenant_id, format, status, total, imported, failed, errors, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		job.ID, job.TenantID, job.Format, job.Status, job.Total, job.Imported, job.Failed,
		"[]", job.CreatedAt,
	)
	return err
}

func (r *importJobRepo) Get(ctx context.Context, id string) (*ImportJob, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, format, status, total, imported, failed, errors, started_at, completed_at, created_at
		 FROM import_jobs WHERE id = $1`, id)

	job := &ImportJob{}
	var errorsJSON []byte
	var startedAt, completedAt *time.Time

	err := row.Scan(&job.ID, &job.TenantID, &job.Format, &job.Status, &job.Total,
		&job.Imported, &job.Failed, &errorsJSON, &startedAt, &completedAt, &job.CreatedAt)
	if err != nil {
		return nil, err
	}

	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	if len(errorsJSON) > 0 && string(errorsJSON) != "[]" {
		_ = json.Unmarshal(errorsJSON, &job.Errors)
	}

	return job, nil
}

func (r *importJobRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*ImportJob, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, format, status, total, imported, failed, errors, started_at, completed_at, created_at
		 FROM import_jobs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*ImportJob
	for rows.Next() {
		job := &ImportJob{}
		var errorsJSON []byte
		var startedAt, completedAt *time.Time

		if err := rows.Scan(&job.ID, &job.TenantID, &job.Format, &job.Status, &job.Total,
			&job.Imported, &job.Failed, &errorsJSON, &startedAt, &completedAt, &job.CreatedAt); err != nil {
			slog.Warn("import job scan error", "error", err)
			continue
		}
		job.StartedAt = startedAt
		job.CompletedAt = completedAt
		if len(errorsJSON) > 0 && string(errorsJSON) != "[]" {
			_ = json.Unmarshal(errorsJSON, &job.Errors)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *importJobRepo) UpdateProgress(ctx context.Context, id string, imported, failed int, errors []ImportRowError) error {
	errorsJSON, _ := json.Marshal(errors)
	_, err := r.pool.Exec(ctx,
		`UPDATE import_jobs SET imported = $2, failed = $3, errors = $4 WHERE id = $1`,
		id, imported, failed, errorsJSON)
	return err
}

func (r *importJobRepo) StartProcessing(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE import_jobs SET status = 'processing', started_at = $2 WHERE id = $1`,
		id, now)
	return err
}

func (r *importJobRepo) Complete(ctx context.Context, id string, imported, failed int, errors []ImportRowError) error {
	now := time.Now().UTC()
	errorsJSON, _ := json.Marshal(errors)
	_, err := r.pool.Exec(ctx,
		`UPDATE import_jobs SET status = 'completed', imported = $2, failed = $3, errors = $4, completed_at = $5 WHERE id = $1`,
		id, imported, failed, errorsJSON, now)
	return err
}

// ImportUserRecord represents a single user to import.
type ImportUserRecord struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// ProcessImportRecords validates and creates users in batches.
// It updates the job progress after each batch.
func (h *HTTPHandler) ProcessImportRecords(ctx context.Context, jobID string, tenantID uuid.UUID, records []ImportUserRecord) {
	repo := h.importJobRepo
	if repo == nil {
		slog.Error("import job repo not configured")
		return
	}

	if err := repo.StartProcessing(ctx, jobID); err != nil {
		slog.Error("failed to mark job as processing", "job_id", jobID, "error", err)
	}

	var importErrors []ImportRowError
	imported := 0
	failed := 0

	for i, rec := range records {
		rowNum := i + 1

		// Validate email format.
		if rec.Email == "" || !isValidEmail(rec.Email) {
			failed++
			importErrors = append(importErrors, ImportRowError{
				Row: rowNum, Username: rec.Username, Error: "invalid or missing email",
			})
			continue
		}

		// Validate username.
		if rec.Username == "" {
			failed++
			importErrors = append(importErrors, ImportRowError{
				Row: rowNum, Error: "missing username",
			})
			continue
		}

		// Validate password strength (min 8 chars).
		if len(rec.Password) < 8 {
			failed++
			importErrors = append(importErrors, ImportRowError{
				Row: rowNum, Username: rec.Username, Error: "password too short (min 8 chars)",
			})
			continue
		}

		// Try to create the user.
		_, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
			TenantID:    tenantID,
			Username:    rec.Username,
			Email:       rec.Email,
			Password:    rec.Password,
			DisplayName: rec.DisplayName,
		})
		if err != nil {
			failed++
			importErrors = append(importErrors, ImportRowError{
				Row: rowNum, Username: rec.Username, Error: fmt.Sprintf("create failed: %v", err),
			})
			continue
		}

		imported++

		// Commit progress every 100 rows.
		if (i+1)%100 == 0 {
			_ = repo.UpdateProgress(ctx, jobID, imported, failed, importErrors)
		}
	}

	// Mark job as complete.
	if err := repo.Complete(ctx, jobID, imported, failed, importErrors); err != nil {
		slog.Error("failed to complete import job", "job_id", jobID, "error", err)
	}

	slog.Info("import job completed",
		"job_id", jobID,
		"total", len(records),
		"imported", imported,
		"failed", failed)
}

// isValidEmail performs a basic email format check.
func isValidEmail(email string) bool {
	at := -1
	dot := -1
	for i, c := range email {
		if c == '@' {
			if at != -1 {
				return false // multiple @
			}
			at = i
		}
		if c == '.' && at != -1 {
			dot = i
		}
	}
	return at > 0 && dot > at+1 && dot < len(email)-1
}
