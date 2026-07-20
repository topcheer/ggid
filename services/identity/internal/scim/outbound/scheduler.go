package outbound

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncSchedule defines how often and what to sync for a target.
type SyncSchedule struct {
	TargetID   string        `json:"target_id"`
	TargetName string        `json:"target_name"`
	Interval   time.Duration `json:"interval"`   // e.g. 5m, 15m, 1h
	Enabled    bool          `json:"enabled"`
	LastSyncAt time.Time     `json:"last_sync_at"`
	NextSyncAt time.Time     `json:"next_sync_at"`
}

// Scheduler periodically pushes GGID users to SCIM targets.
type Scheduler struct {
	client    *Client
	pool      *pgxpool.Pool
	schedules map[string]*SyncSchedule // target name → schedule
	stopCh    chan struct{}
}

// NewScheduler creates a SCIM outbound sync scheduler.
func NewScheduler(client *Client, pool *pgxpool.Pool) *Scheduler {
	return &Scheduler{
		client:    client,
		pool:      pool,
		schedules: make(map[string]*SyncSchedule),
		stopCh:    make(chan struct{}),
	}
}

// EnsureSchema creates the schedule table.
func (s *Scheduler) EnsureSchema(ctx context.Context) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS scim_sync_schedules (
			target_name TEXT PRIMARY KEY,
			target_id TEXT NOT NULL,
			interval_seconds INT NOT NULL DEFAULT 300,
			enabled BOOLEAN NOT NULL DEFAULT true,
			last_sync_at TIMESTAMPTZ,
			next_sync_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

// SetSchedule configures a sync schedule for a target.
func (s *Scheduler) SetSchedule(ctx context.Context, schedule *SyncSchedule) error {
	if s.pool != nil {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO scim_sync_schedules (target_name, target_id, interval_seconds, enabled, next_sync_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, now())
			 ON CONFLICT (target_name) DO UPDATE SET
			   target_id = EXCLUDED.target_id,
			   interval_seconds = EXCLUDED.interval_seconds,
			   enabled = EXCLUDED.enabled,
			   next_sync_at = EXCLUDED.next_sync_at,
			   updated_at = now()`,
			schedule.TargetName, schedule.TargetID, int(schedule.Interval.Seconds()),
			schedule.Enabled, schedule.NextSyncAt)
		if err != nil {
			return err
		}
	}
	s.schedules[schedule.TargetName] = schedule
	return nil
}

// ListSchedules returns all configured sync schedules.
func (s *Scheduler) ListSchedules(ctx context.Context) ([]*SyncSchedule, error) {
	if s.pool == nil {
		var result []*SyncSchedule
		for _, sched := range s.schedules {
			result = append(result, sched)
		}
		return result, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT target_name, target_id, interval_seconds, enabled,
		        COALESCE(last_sync_at, 'epoch'), COALESCE(next_sync_at, 'epoch')
		 FROM scim_sync_schedules ORDER BY target_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*SyncSchedule
	for rows.Next() {
		var sched SyncSchedule
		var intervalSecs int
		if err := rows.Scan(&sched.TargetName, &sched.TargetID, &intervalSecs,
			&sched.Enabled, &sched.LastSyncAt, &sched.NextSyncAt); err != nil {
			continue
		}
		sched.Interval = time.Duration(intervalSecs) * time.Second
		result = append(result, &sched)
	}
	return result, nil
}

// Start begins the scheduler loop. It runs until Stop is called.
// The ticker interval controls how often schedules are checked.
func (s *Scheduler) Start(ctx context.Context, checkInterval time.Duration) {
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	slog.Info("SCIM outbound scheduler started", "check_interval", checkInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("SCIM outbound scheduler stopped (context cancelled)")
			return
		case <-s.stopCh:
			slog.Info("SCIM outbound scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// Stop signals the scheduler to exit.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// tick checks all schedules and runs due syncs.
func (s *Scheduler) tick(ctx context.Context) {
	schedules, err := s.ListSchedules(ctx)
	if err != nil {
		slog.Warn("SCIM scheduler: failed to list schedules", "error", err)
		return
	}

	now := time.Now()
	for _, sched := range schedules {
		if !sched.Enabled {
			continue
		}
		if sched.NextSyncAt.IsZero() || now.After(sched.NextSyncAt) {
			slog.Info("SCIM scheduler: syncing target",
				"target", sched.TargetName, "target_id", sched.TargetID)
			if err := s.syncTarget(ctx, sched); err != nil {
				slog.Warn("SCIM scheduler: sync failed",
					"target", sched.TargetName, "error", err)
			}
			sched.LastSyncAt = now
			sched.NextSyncAt = now.Add(sched.Interval)
			s.updateScheduleStatus(ctx, sched)
		}
	}
}

// syncTarget performs a full user sync to one SCIM target.
func (s *Scheduler) syncTarget(ctx context.Context, sched *SyncSchedule) error {
	if s.client == nil {
		return fmt.Errorf("SCIM client not configured")
	}

	// Fetch users that need syncing from the GGID users table.
	// In production this queries the identity service; here we fetch from DB directly.
	if s.pool == nil {
		return nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, username, COALESCE(display_name, username), COALESCE(email, ''), active
		 FROM users WHERE tenant_id IS NOT NULL LIMIT 500`)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	synced := 0
	failed := 0
	for rows.Next() {
		var user GGIDUser
		if err := rows.Scan(&user.ID, &user.UserName, &user.DisplayName, &user.Email, &user.Active); err != nil {
			continue
		}

		// Determine operation: create or update.
		// For simplicity, we try create first; if user exists (409), we update.
		op := OpCreateUser
		logEntry, err := s.client.Execute(ctx, sched.TargetName, op, user)
		if err != nil && logEntry != nil && logEntry.Error != "" {
			// Try update if create failed (user might already exist)
			_, updateErr := s.client.Execute(ctx, sched.TargetName, OpUpdateUser, user)
			if updateErr != nil {
				failed++
				continue
			}
		}
		synced++
	}

	slog.Info("SCIM sync complete",
		"target", sched.TargetName,
		"synced", synced, "failed", failed)
	return nil
}

// updateScheduleStatus persists last_sync_at and next_sync_at.
func (s *Scheduler) updateScheduleStatus(ctx context.Context, sched *SyncSchedule) {
	if s.pool == nil {
		return
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE scim_sync_schedules SET last_sync_at = $1, next_sync_at = $2, updated_at = now()
		 WHERE target_name = $3`,
		sched.LastSyncAt, sched.NextSyncAt, sched.TargetName)
	if err != nil {
		slog.Warn("SCIM scheduler: failed to update schedule status", "error", err)
	}
}
