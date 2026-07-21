

// repair_audit_chain.go — one-time migration: re-compute the audit_events
// hash chain over the CURRENT stored column values.
//
// Background: before the Insert fix (Task-G), hashes were computed over
// uuid.Nil ID and a zero CreatedAt (DB generated those after hashing), so
// stored hashes were never reproducible from stored data. This tool
// re-chains every tenant's events so the chain becomes verifiable; future
// tampering is then detectable by /api/v1/audit/tamper-check.
//
// Usage:
//
//	kubectl port-forward -n ggid svc/ggid-postgresql 15432:5432 &
//	AUDIT_HASH_CHAIN_SECRET=<secret> go run ./services/audit/cmd/repair-chain
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	secret := os.Getenv("AUDIT_HASH_CHAIN_SECRET")
	if secret == "" {
		log.Fatal("AUDIT_HASH_CHAIN_SECRET required")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://ggid:ggid-k3s@localhost:15432/ggid"
	}
	domain.SetHashChainSecret([]byte(secret))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	tenants, err := pool.Query(ctx, `SELECT DISTINCT tenant_id FROM audit_events`)
	if err != nil {
		log.Fatalf("list tenants: %v", err)
	}
	var tenantIDs []uuid.UUID
	for tenants.Next() {
		var id uuid.UUID
		if err := tenants.Scan(&id); err == nil {
			tenantIDs = append(tenantIDs, id)
		}
	}
	tenants.Close()

	for _, tid := range tenantIDs {
		updated, total, err := repairTenant(ctx, pool, tid)
		if err != nil {
			log.Fatalf("tenant %s: %v", tid, err)
		}
		fmt.Printf("tenant %s: %d/%d events re-chained\n", tid, updated, total)
	}
	fmt.Println("chain repair complete")
}

func repairTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) (int, int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, tenant_id, actor_type, actor_id, action,
		       COALESCE(resource_type, ''), resource_id, result,
		       COALESCE(ip_address::text, ''),
		       COALESCE(prev_hash, ''), COALESCE(hash, ''), created_at
		FROM audit_events WHERE tenant_id = $1
		ORDER BY created_at ASC, id ASC`, tenantID)
	if err != nil {
		return 0, 0, err
	}
	var events []*domain.AuditEvent
	for rows.Next() {
		e := &domain.AuditEvent{}
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ActorType, &e.ActorID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Result, &e.IPAddress,
			&e.PrevHash, &e.Hash, &e.CreatedAt); err != nil {
			rows.Close()
			return 0, 0, err
		}
		events = append(events, e)
	}
	rows.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL app.allow_audit_mutation = 'on'`); err != nil {
		return 0, 0, fmt.Errorf("set GUC: %w", err)
	}

	prevHash := ""
	updated := 0
	for _, e := range events {
		e.CreatedAt = e.CreatedAt.UTC().Truncate(time.Microsecond)
		newHash := e.ComputeHash(prevHash)
		if e.PrevHash != prevHash || e.Hash != newHash {
			if _, err := tx.Exec(ctx,
				`UPDATE audit_events SET prev_hash = $1, hash = $2 WHERE id = $3`,
				prevHash, newHash, e.ID); err != nil {
				return 0, 0, fmt.Errorf("update %s: %w", e.ID, err)
			}
			updated++
		}
		prevHash = newHash
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return updated, len(events), nil
}
