package scim

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// groups_store.go — Task-C: PostgreSQL persistence for SCIM Groups.
// Previously groups were created/updated only in process memory (or not at
// all), so Pod restarts lost them and replicas diverged. API shape is
// unchanged; only the storage layer is replaced.

var groupSchemaOnce sync.Once

// ensureGroupSchema creates the scim_groups table exactly once per process.
func ensureGroupSchema(ctx context.Context, pool *pgxpool.Pool) {
	groupSchemaOnce.Do(func() {
		_, _ = pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS scim_groups (
				id           TEXT PRIMARY KEY,
				tenant_id    UUID,
				display_name TEXT NOT NULL,
				members      JSONB NOT NULL DEFAULT '[]',
				created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
				updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
			);
			CREATE INDEX IF NOT EXISTS idx_scim_groups_tenant ON scim_groups(tenant_id);
		`)
	})
}

// dbCreateGroup inserts a new SCIM group.
func dbCreateGroup(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, g *SCIMGroup) error {
	members, err := json.Marshal(g.Members)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO scim_groups (id, tenant_id, display_name, members)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name,
			members = EXCLUDED.members, updated_at = now()
	`, g.ID, tenantID, g.DisplayName, members)
	return err
}

// dbGetGroup loads a SCIM group by ID. Returns nil when not found.
func dbGetGroup(ctx context.Context, pool *pgxpool.Pool, id string) (*SCIMGroup, error) {
	var (
		displayName string
		membersRaw  []byte
	)
	err := pool.QueryRow(ctx, `
		SELECT display_name, members FROM scim_groups WHERE id = $1
	`, id).Scan(&displayName, &membersRaw)
	if err != nil {
		return nil, err
	}
	g := &SCIMGroup{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		ID:          id,
		DisplayName: displayName,
		Meta:        SCIMMeta{ResourceType: "Group", Location: "/scim/v2/Groups/" + id},
	}
	_ = json.Unmarshal(membersRaw, &g.Members)
	return g, nil
}

// dbListGroups returns all persisted SCIM groups for a tenant (uuid.Nil
// tenant lists all, preserving the previous tenant-agnostic behavior).
func dbListGroups(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) ([]SCIMGroup, error) {
	query := `SELECT id, display_name, members FROM scim_groups`
	args := []any{}
	if tenantID != uuid.Nil {
		query += ` WHERE tenant_id = $1`
		args = append(args, tenantID)
	}
	query += ` ORDER BY display_name`
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SCIMGroup
	for rows.Next() {
		var (
			id, name   string
			membersRaw []byte
		)
		if err := rows.Scan(&id, &name, &membersRaw); err != nil {
			continue
		}
		g := SCIMGroup{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			ID:          id,
			DisplayName: name,
			Meta:        SCIMMeta{ResourceType: "Group", Location: "/scim/v2/Groups/" + id},
		}
		_ = json.Unmarshal(membersRaw, &g.Members)
		out = append(out, g)
	}
	return out, rows.Err()
}

// dbUpdateGroup persists display name + members for an existing group.
func dbUpdateGroup(ctx context.Context, pool *pgxpool.Pool, g *SCIMGroup) error {
	members, err := json.Marshal(g.Members)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE scim_groups SET display_name = $2, members = $3, updated_at = $4
		WHERE id = $1
	`, g.ID, g.DisplayName, members, time.Now())
	return err
}

// dbDeleteGroup removes a SCIM group by ID.
func dbDeleteGroup(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM scim_groups WHERE id = $1`, id)
	return err
}
