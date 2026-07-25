package main

import (
	"context"
	"fmt"
	"time"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	domain.SetHashChainSecret([]byte("ee24b9b1b112a0c835dcaad305ab32be5d590f5d3ece9f6b4f1930913cbd1357"))
	pool, _ := pgxpool.New(context.Background(), "postgres://ggid:ggid-k3s@localhost:5432/ggid?sslmode=disable")
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
