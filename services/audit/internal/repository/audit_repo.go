package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepository manages audit event persistence and queries.
type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

// Insert writes a single audit event to the database.
// It computes the hash chain link using the previous event's hash.
// The event ID and CreatedAt are assigned here (before hashing) so the
// stored hash is reproducible from the persisted column values.
func (r *AuditRepository) Insert(ctx context.Context, e *domain.AuditEvent) error {
	metaJSON, _ := json.Marshal(e.Metadata)
	var ipAddr any
	if e.IPAddress != "" {
		// Strip port if present (inet type doesn't accept IP:PORT format)
		ip := e.IPAddress
		if idx := strings.LastIndex(ip, ":"); idx > 0 && !strings.Contains(ip[idx+1:], ":") {
			ip = ip[:idx] // IPv4:port → IPv4
		}
		ipAddr = ip
	}

	// Assign ID and timestamp in Go (before hashing) instead of relying on
	// DB defaults — the hash chain must be computed over the exact values
	// that get persisted. Truncate to microseconds: timestamptz stores µs
	// precision, and verification recomputes the hash from DB values.
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	e.CreatedAt = e.CreatedAt.UTC().Truncate(time.Microsecond)

	// Use a transaction with FOR UPDATE to prevent the TOCTOU race condition:
	// without a lock, two concurrent inserts can read the same PrevHash,
	// producing a broken chain link.
	tx, txErr := r.db.Begin(ctx)
	if txErr != nil {
		return fmt.Errorf("begin tx: %w", txErr)
	}
	defer tx.Rollback(ctx)

	if domain.IsHashChainEnabled() {
		var ph string
		// FOR UPDATE locks the row, serializing chain appends per tenant.
		err := tx.QueryRow(ctx,
			`SELECT COALESCE(hash, '') FROM audit_events
			 WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE`,
			e.TenantID,
		).Scan(&ph)
		if err != nil && err != pgx.ErrNoRows {
			return fmt.Errorf("query prev hash: %w", err)
		}
		e.PrevHash = ph
		e.Hash = e.ComputeHash(ph)
	}

	query := `
		INSERT INTO audit_events (id, tenant_id, actor_type, actor_id, actor_name, action,
		    resource_type, resource_id, resource_name, result, ip_address,
		    user_agent, request_id, metadata, prev_hash, hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::inet, $12, $13, $14, $15, $16, $17)`
	_, err := tx.Exec(ctx, query,
		e.ID, e.TenantID, e.ActorType, nullableUUID(e.ActorID), nullableStr(e.ActorName), e.Action,
		nullableStr(e.ResourceType), nullableUUID(e.ResourceID), nullableStr(e.ResourceName), e.Result, ipAddr,
		nullableStr(e.UserAgent), nullableStr(e.RequestID), metaJSON,
		e.PrevHash, e.Hash, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return tx.Commit(ctx)
}

// nullableStr returns nil for empty strings so PostgreSQL stores NULL.
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableUUID returns nil for nil pointers and uuid.Nil so PostgreSQL stores NULL.
func nullableUUID(u *uuid.UUID) any {
	if u == nil || *u == uuid.Nil {
		return nil
	}
	return u
}

// GetByID retrieves a single audit event by ID.
func (r *AuditRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AuditEvent, error) {
	event := &domain.AuditEvent{}
	var metaBytes []byte
	var actorName, resourceType, resourceName, ipAddr, userAgent, requestID *string
	query := `
		SELECT id, tenant_id, actor_type, actor_id, actor_name, action,
		    resource_type, resource_id, resource_name, result,
			    host(ip_address), user_agent, request_id, metadata,
			    COALESCE(prev_hash, ''), COALESCE(hash, ''),
			    created_at
		    FROM audit_events WHERE id = $1 AND tenant_id = $2`
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&event.ID, &event.TenantID, &event.ActorType, &event.ActorID, &actorName,
		&event.Action, &resourceType, &event.ResourceID, &resourceName,
		&event.Result, &ipAddr, &userAgent, &requestID, &metaBytes,
		&event.PrevHash, &event.Hash, &event.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err, "audit_event", id.String())
	}
	if len(metaBytes) > 0 {
		json.Unmarshal(metaBytes, &event.Metadata)
	}
	event.ActorName = ptrStr(actorName)
	event.ResourceType = ptrStr(resourceType)
	event.ResourceName = ptrStr(resourceName)
	event.IPAddress = ptrStr(ipAddr)
	event.UserAgent = ptrStr(userAgent)
	event.RequestID = ptrStr(requestID)
	return event, nil
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// List returns audit events matching the filter with pagination.
func (r *AuditRepository) List(ctx context.Context, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error) {
	where := "WHERE tenant_id = $1"
	args := []any{filter.TenantID}
	n := 2

	if filter.ActorID != nil {
		where += fmt.Sprintf(" AND actor_id = $%d", n)
		args = append(args, *filter.ActorID)
		n++
	}
	if filter.Action != "" {
		where += fmt.Sprintf(" AND action LIKE $%d ESCAPE '\\'", n)
		args = append(args, "%"+escapeLikeWildcards(filter.Action)+"%")
		n++
	}
	if filter.ResourceType != "" {
		where += fmt.Sprintf(" AND resource_type = $%d", n)
		args = append(args, filter.ResourceType)
		n++
	}
	if filter.Result != "" {
		where += fmt.Sprintf(" AND result = $%d", n)
		args = append(args, filter.Result)
		n++
	}
	if filter.StartTime != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", n)
		args = append(args, *filter.StartTime)
		n++
	}
	if filter.EndTime != nil {
		where += fmt.Sprintf(" AND created_at < $%d", n)
		args = append(args, *filter.EndTime)
		n++
	}

	// Count total
	countQuery := "SELECT count(*) FROM audit_events " + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}

	// Build ORDER BY
	orderCol := "created_at"
	switch filter.OrderBy {
	case "action":
		orderCol = "action"
	case "actor_name":
		orderCol = "actor_name"
	}
	orderDir := "DESC"
	if !filter.Descending {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, actor_type, actor_id, actor_name, action,
		    resource_type, resource_id, resource_name, result,
			    host(ip_address), user_agent, request_id, metadata, created_at
		FROM audit_events %s
		ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, orderCol, orderDir, n, n+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		e := &domain.AuditEvent{}
		var metaBytes []byte
		var actorName, resourceType, resourceName, ipAddr, userAgent, requestID *string
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ActorType, &e.ActorID, &actorName, &e.Action,
			&resourceType, &e.ResourceID, &resourceName, &e.Result,
			&ipAddr, &userAgent, &requestID, &metaBytes, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(metaBytes) > 0 {
			json.Unmarshal(metaBytes, &e.Metadata)
		}
		e.ActorName = ptrStr(actorName)
		e.ResourceType = ptrStr(resourceType)
		e.ResourceName = ptrStr(resourceName)
		e.IPAddress = ptrStr(ipAddr)
		e.UserAgent = ptrStr(userAgent)
		e.RequestID = ptrStr(requestID)
		events = append(events, e)
	}
	return events, total, nil
}

// GetStats returns aggregated audit statistics for the given tenant since the given time.
func (r *AuditRepository) GetStats(ctx context.Context, tenantID uuid.UUID, since time.Time) (*domain.Stats, error) {
	stats := &domain.Stats{
		EventsByAction: make(map[string]int),
	}

	// 1. Total events in last 24h
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM audit_events WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, since,
	).Scan(&stats.TotalEvents24h); err != nil {
		return nil, fmt.Errorf("count total events: %w", err)
	}

	// 2. Count by action
	rows, err := r.db.Query(ctx,
		`SELECT action, count(*) FROM audit_events
		 WHERE tenant_id = $1 AND created_at >= $2
		 GROUP BY action ORDER BY count(*) DESC`,
		tenantID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("query events by action: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		stats.EventsByAction[action] = count
	}

	// 3. Hourly distribution (24 buckets)
	hourlyRows, err := r.db.Query(ctx,
		`SELECT date_trunc('hour', created_at) AS hour, count(*) AS cnt
		 FROM audit_events
		 WHERE tenant_id = $1 AND created_at >= $2
		 GROUP BY hour ORDER BY hour`,
		tenantID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("query hourly distribution: %w", err)
	}
	defer hourlyRows.Close()
	for hourlyRows.Next() {
		var hc domain.HourlyCount
		if err := hourlyRows.Scan(&hc.Hour, &hc.Count); err != nil {
			return nil, err
		}
		stats.HourlyDistribution = append(stats.HourlyDistribution, hc)
	}

	// 4. Top 10 active actors — use actor_name from audit_events directly
	// (LEFT JOIN users removed because RLS on users table blocks cross-service queries)
	actorRows, err := r.db.Query(ctx,
		`SELECT ae.actor_id,
		        COALESCE(ae.actor_name, CASE WHEN ae.actor_id = '00000000-0000-0000-0000-000000000000' THEN 'system' ELSE 'unknown' END) AS display_name,
		        count(*) AS cnt
		 FROM audit_events ae
		 WHERE ae.tenant_id = $1 AND ae.created_at >= $2 AND ae.actor_id IS NOT NULL
		 GROUP BY ae.actor_id, display_name
		 ORDER BY cnt DESC LIMIT 10`,
		tenantID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("query top actors: %w", err)
	}
	defer actorRows.Close()
	for actorRows.Next() {
		var actorIDStr string
		var actorName string
		var count int
		if err := actorRows.Scan(&actorIDStr, &actorName, &count); err != nil {
			return nil, err
		}
		aa := domain.ActorActivity{
			ActorName: actorName,
			Count:     count,
		}
		if parsed, err := uuid.Parse(actorIDStr); err == nil {
			aa.ActorID = parsed
		}
		stats.TopActors = append(stats.TopActors, aa)
	}

	// 5. Failed logins in 24h
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		 WHERE tenant_id = $1 AND created_at >= $2
		   AND action = 'user.login' AND result = 'failure'`,
		tenantID, since,
	).Scan(&stats.FailedLogins24h); err != nil {
		return nil, fmt.Errorf("count failed logins: %w", err)
	}

	return stats, nil
}

// DeleteOlderThan deletes audit events older than the given time.
// Returns the number of deleted rows. Authorized retention deletions set the
// app.allow_audit_mutation GUC to bypass the WORM trigger inside the tx.
func (r *AuditRepository) DeleteOlderThan(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retention tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL app.allow_audit_mutation = 'on'`); err != nil {
		return 0, fmt.Errorf("allow audit mutation: %w", err)
	}
	// Tenant-scoped delete. A NULL tenant param means "all tenants" and is
	// reserved for the internal scheduled retention job — the HTTP path
	// must always pass a real tenant (P0: cross-tenant audit destruction).
	var tid any
	if tenantID != uuid.Nil {
		tid = tenantID
	}
	tag, err := tx.Exec(ctx,
		`DELETE FROM audit_events WHERE created_at < $1 AND ($2::uuid IS NULL OR tenant_id = $2)`,
		before, tid,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old audit events: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit retention tx: %w", err)
	}
	return tag.RowsAffected(), nil
}

// GetBoundaryEventID returns the newest event ID older than 'before' for
// hash chain continuity preservation during retention cleanup.
func (r *AuditRepository) GetBoundaryEventID(ctx context.Context, tenantID uuid.UUID, before time.Time) (uuid.UUID, error) {
	var tid any
	if tenantID != uuid.Nil {
		tid = tenantID
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM audit_events
		 WHERE created_at < $1 AND ($2::uuid IS NULL OR tenant_id = $2)
		 ORDER BY created_at DESC LIMIT 1`,
		before, tid).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// DeleteOlderThanExcept deletes events older than 'before' except the event
// with the given exceptID (preserved as chain anchor for integrity).
func (r *AuditRepository) DeleteOlderThanExcept(ctx context.Context, tenantID uuid.UUID, before time.Time, exceptID uuid.UUID) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retention tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL app.allow_audit_mutation = 'on'`); err != nil {
		return 0, fmt.Errorf("allow audit mutation: %w", err)
	}
	var tid any
	if tenantID != uuid.Nil {
		tid = tenantID
	}
	tag, err := tx.Exec(ctx,
		`DELETE FROM audit_events WHERE created_at < $1 AND id != $3 AND ($2::uuid IS NULL OR tenant_id = $2)`,
		before, tid, exceptID,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old audit events: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit retention tx: %w", err)
	}
	return tag.RowsAffected(), nil
}

// AnonymizeOlderThan nullifies PII fields (actor_ip, user_agent, metadata)
// for events older than the cutoff time. Keeps the event row for compliance.
func (r *AuditRepository) AnonymizeOlderThan(ctx context.Context, tenantID *uuid.UUID, before time.Time) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin anonymize tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL app.allow_audit_mutation = 'on'`); err != nil {
		return 0, fmt.Errorf("allow audit mutation: %w", err)
	}
	// SECURITY: Optional tenant_id filter. If tenantID is nil, anonymizes across
	// all tenants (platform-level compliance operation). If set, only affects
	// the specified tenant's events.
	var query string
	var args []any
	if tenantID != nil {
		query = `UPDATE audit_events SET actor_ip = NULL, user_agent = NULL, metadata = '{}'::jsonb WHERE created_at < $1 AND tenant_id = $2`
		args = []any{before, *tenantID}
	} else {
		query = `UPDATE audit_events SET actor_ip = NULL, user_agent = NULL, metadata = '{}'::jsonb WHERE created_at < $1`
		args = []any{before}
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("anonymize old audit events: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit anonymize tx: %w", err)
	}
	return tag.RowsAffected(), nil
}

func mapErr(err error, resource, id string) error {
	if err == pgx.ErrNoRows {
		return errors.NotFound(resource, id)
	}
	return errors.Wrap(errors.ErrInternal, "database error", err)
}

// escapeLikeWildcards escapes LIKE wildcard characters (% and _) and the
// escape character backslash in user-supplied search input to prevent
// pattern injection that could cause information leakage or DoS.
func escapeLikeWildcards(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
