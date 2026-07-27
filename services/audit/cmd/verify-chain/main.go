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
		log.Fatal("DATABASE_URL required")
	}
	domain.SetHashChainSecret([]byte(secret))
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	rows, _ := pool.Query(context.Background(), "SELECT DISTINCT tenant_id FROM audit_events")
	var tenants []uuid.UUID
	for rows.Next() { var t uuid.UUID; rows.Scan(&t); tenants = append(tenants, t) }
	rows.Close()

	totalBroken := 0
	for _, tid := range tenants {
		evRows, _ := pool.Query(context.Background(), `
			SELECT id, tenant_id, actor_type, actor_id, action,
			       COALESCE(resource_type,''), COALESCE(resource_id::text,''), result,
			       COALESCE(ip_address::text,''), COALESCE(prev_hash,''), COALESCE(hash,''), created_at
			FROM audit_events WHERE tenant_id = $1 ORDER BY created_at ASC, id ASC
		`, tid)
		prevHash := ""
		tenantBroken := 0
		for evRows.Next() {
			e := &domain.AuditEvent{}
			evRows.Scan(&e.ID, &e.TenantID, &e.ActorType, &e.ActorID, &e.Action,
				&e.ResourceType, &e.ResourceID, &e.Result, &e.IPAddress,
				&e.PrevHash, &e.Hash, &e.CreatedAt)
			e.CreatedAt = e.CreatedAt.UTC().Truncate(time.Microsecond)
			computed := e.ComputeHash(prevHash)
			if e.Hash != computed || e.PrevHash != prevHash { tenantBroken++ }
			prevHash = computed
		}
		evRows.Close()
		if tenantBroken > 0 { fmt.Printf("tenant %s: %d broken\n", tid.String()[:8], tenantBroken) }
		totalBroken += tenantBroken
	}
	fmt.Printf("Total broken chains: %d\n", totalBroken)
}
