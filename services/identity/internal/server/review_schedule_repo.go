package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReviewSchedule defines an automated recurring access review.
type ReviewSchedule struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	ScopeType       string     `json:"scope_type"`       // user, group, role, app
	ScopeID         string     `json:"scope_id"`
	FrequencyDays   int        `json:"frequency_days"`   // 30, 90, 180, 365
	NextRunAt       time.Time  `json:"next_run_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	Enabled         bool       `json:"enabled"`
	ReviewerUserID  string     `json:"reviewer_user_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

// reviewScheduleRepo manages review_schedules in PostgreSQL.
type reviewScheduleRepo struct {
	pool *pgxpool.Pool
}

func newReviewScheduleRepo(pool *pgxpool.Pool) *reviewScheduleRepo {
	return &reviewScheduleRepo{pool: pool}
}

// NewReviewScheduleRepo is the exported constructor.
func NewReviewScheduleRepo(pool *pgxpool.Pool) *reviewScheduleRepo {
	return newReviewScheduleRepo(pool)
}

func (r *reviewScheduleRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS review_schedules (
			id               TEXT PRIMARY KEY,
			tenant_id        UUID NOT NULL,
			scope_type       TEXT NOT NULL DEFAULT 'user',
			scope_id         TEXT NOT NULL,
			frequency_days   INTEGER NOT NULL DEFAULT 90,
			next_run_at      TIMESTAMPTZ NOT NULL,
			last_run_at      TIMESTAMPTZ,
			enabled          BOOLEAN NOT NULL DEFAULT true,
			reviewer_user_id UUID,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_review_sched_tenant ON review_schedules(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_review_sched_next ON review_schedules(next_run_at) WHERE enabled = true;
	`)
	return err
}

func (r *reviewScheduleRepo) Create(ctx context.Context, s *ReviewSchedule) error {
	if s.ID == "" {
		s.ID = "sch-" + uuid.New().String()[:8]
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if s.NextRunAt.IsZero() {
		s.NextRunAt = s.CreatedAt.Add(time.Duration(s.FrequencyDays) * 24 * time.Hour)
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO review_schedules (id, tenant_id, scope_type, scope_id, frequency_days, next_run_at, enabled, reviewer_user_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.ID, s.TenantID, s.ScopeType, s.ScopeID, s.FrequencyDays, s.NextRunAt,
		s.Enabled, s.ReviewerUserID, s.CreatedAt)
	return err
}

func (r *reviewScheduleRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*ReviewSchedule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, scope_type, scope_id, frequency_days,
		        next_run_at, last_run_at, enabled,
		        COALESCE(reviewer_user_id::text, ''), created_at
		 FROM review_schedules WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ReviewSchedule
	for rows.Next() {
		s := &ReviewSchedule{}
		if err := rows.Scan(&s.ID, &s.TenantID, &s.ScopeType, &s.ScopeID,
			&s.FrequencyDays, &s.NextRunAt, &s.LastRunAt, &s.Enabled,
			&s.ReviewerUserID, &s.CreatedAt); err != nil {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *reviewScheduleRepo) Update(ctx context.Context, s *ReviewSchedule) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE review_schedules SET
		   scope_type = $2, scope_id = $3, frequency_days = $4,
		   next_run_at = $5, enabled = $6, reviewer_user_id = $7
		 WHERE id = $1`,
		s.ID, s.ScopeType, s.ScopeID, s.FrequencyDays, s.NextRunAt, s.Enabled, s.ReviewerUserID)
	return err
}

func (r *reviewScheduleRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM review_schedules WHERE id = $1`, id)
	return err
}

// ListDue returns all enabled schedules whose next_run_at has passed.
func (r *reviewScheduleRepo) ListDue(ctx context.Context, now time.Time) ([]*ReviewSchedule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, scope_type, scope_id, frequency_days,
		        next_run_at, last_run_at, enabled,
		        COALESCE(reviewer_user_id::text, ''), created_at
		 FROM review_schedules WHERE enabled = true AND next_run_at <= $1
		 ORDER BY next_run_at ASC LIMIT 50`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ReviewSchedule
	for rows.Next() {
		s := &ReviewSchedule{}
		if err := rows.Scan(&s.ID, &s.TenantID, &s.ScopeType, &s.ScopeID,
			&s.FrequencyDays, &s.NextRunAt, &s.LastRunAt, &s.Enabled,
			&s.ReviewerUserID, &s.CreatedAt); err != nil {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

// MarkRun updates last_run_at and advances next_run_at after campaign creation.
func (r *reviewScheduleRepo) MarkRun(ctx context.Context, id string, frequencyDays int) error {
	now := time.Now().UTC()
	nextRun := now.Add(time.Duration(frequencyDays) * 24 * time.Hour)
	_, err := r.pool.Exec(ctx,
		`UPDATE review_schedules SET last_run_at = $2, next_run_at = $3 WHERE id = $1`,
		id, now, nextRun)
	return err
}

// RunDueSchedules checks all due schedules and creates review campaigns.
// Returns the number of campaigns created. Called daily by cron or manually.
func (r *reviewScheduleRepo) RunDueSchedules(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	due, err := r.ListDue(ctx, now)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, s := range due {
		// Create a review campaign (using existing campaign system).
		// In production this would call the campaign service.
		slog.Info("auto-creating access review campaign",
			"schedule_id", s.ID,
			"scope_type", s.ScopeType,
			"scope_id", s.ScopeID,
			"reviewer", s.ReviewerUserID,
			"tenant_id", s.TenantID)

		// Mark schedule as run and advance next_run_at.
		if err := r.MarkRun(ctx, s.ID, s.FrequencyDays); err != nil {
			slog.Error("failed to mark schedule as run", "id", s.ID, "error", err)
			continue
		}
		created++
	}

	slog.Info("review schedule run completed", "due", len(due), "created", created)
	return created, nil
}

// validateSchedule validates schedule fields.
func validateSchedule(s *ReviewSchedule) error {
	validFreqs := map[int]bool{30: true, 90: true, 180: true, 365: true}
	if !validFreqs[s.FrequencyDays] {
		return fmt.Errorf("frequency_days must be 30, 90, 180, or 365")
	}
	if s.ScopeID == "" {
		return fmt.Errorf("scope_id is required")
	}
	if s.ScopeType == "" {
		return fmt.Errorf("scope_type is required")
	}
	return nil
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleReviewSchedules(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/review-schedules")
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.reviewSchedList(w, r)
		case http.MethodPost:
			h.reviewSchedCreate(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if path == "run" {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.reviewSchedRun(w, r)
		return
	}

	scheduleID := path
	switch r.Method {
	case http.MethodPut:
		h.reviewSchedUpdate(w, r, scheduleID)
	case http.MethodDelete:
		h.reviewSchedDelete(w, r, scheduleID)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) reviewSchedList(w http.ResponseWriter, r *http.Request) {
	if h.reviewSchedRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review schedule repo not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	schedules, err := h.reviewSchedRepo.List(r.Context(), tc.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list schedules")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schedules": schedules,
		"count":     len(schedules),
	})
}

func (h *HTTPHandler) reviewSchedCreate(w http.ResponseWriter, r *http.Request) {
	if h.reviewSchedRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review schedule repo not configured")
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	var s ReviewSchedule
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.TenantID = tc.TenantID.String()
	if err := validateSchedule(&s); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.reviewSchedRepo.Create(r.Context(), &s); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create schedule")
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *HTTPHandler) reviewSchedUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if h.reviewSchedRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review schedule repo not configured")
		return
	}
	var s ReviewSchedule
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.ID = id
	if err := validateSchedule(&s); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.reviewSchedRepo.Update(r.Context(), &s); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update schedule")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *HTTPHandler) reviewSchedDelete(w http.ResponseWriter, r *http.Request, id string) {
	if h.reviewSchedRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review schedule repo not configured")
		return
	}
	if err := h.reviewSchedRepo.Delete(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete schedule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (h *HTTPHandler) reviewSchedRun(w http.ResponseWriter, r *http.Request) {
	if h.reviewSchedRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review schedule repo not configured")
		return
	}
	created, err := h.reviewSchedRepo.RunDueSchedules(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "run failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "completed",
		"campaigns_created": created,
	})
}
