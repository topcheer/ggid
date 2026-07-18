package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttributeMapping defines how legacy system attributes/groups map to GGID roles.
type AttributeMapping struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	SourceType     string `json:"source_type"`     // ldap, scim, custom_db, csv
	SourceAttribute string `json:"source_attribute"` // e.g. "memberOf", "department"
	SourceValue    string `json:"source_value"`     // e.g. "CN=Admins,OU=Groups"
	GGIDRole       string `json:"ggid_role"`        // e.g. "admin", "developer", "viewer"
	GGIDAttribute  string `json:"ggid_attribute,omitempty"` // optional: map to a custom attribute
	Priority       int    `json:"priority"`         // higher = evaluated first
	Enabled        bool   `json:"enabled"`
}

// attributeMappingRepo manages attribute_mappings in PostgreSQL.
type attributeMappingRepo struct {
	pool *pgxpool.Pool
}

// NewAttributeMappingRepo creates a new attribute mapping repository (exported).
func NewAttributeMappingRepo(pool *pgxpool.Pool) *attributeMappingRepo {
	return newAttributeMappingRepo(pool)
}

func newAttributeMappingRepo(pool *pgxpool.Pool) *attributeMappingRepo {
	return &attributeMappingRepo{pool: pool}
}

func (r *attributeMappingRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS attribute_mappings (
			id               TEXT PRIMARY KEY,
			tenant_id        UUID NOT NULL,
			source_type      TEXT NOT NULL DEFAULT 'ldap',
			source_attribute TEXT NOT NULL,
			source_value     TEXT NOT NULL,
			ggid_role        TEXT NOT NULL,
			ggid_attribute   TEXT DEFAULT '',
			priority         INTEGER NOT NULL DEFAULT 0,
			enabled          BOOLEAN NOT NULL DEFAULT true,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_attr_mappings_tenant ON attribute_mappings(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_attr_mappings_source ON attribute_mappings(source_type, source_attribute);
	`)
	return err
}

func (r *attributeMappingRepo) List(ctx context.Context, tenantID uuid.UUID) ([]AttributeMapping, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, source_type, source_attribute, source_value,
		        ggid_role, ggid_attribute, priority, enabled
		 FROM attribute_mappings WHERE tenant_id = $1 ORDER BY priority DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AttributeMapping
	for rows.Next() {
		var m AttributeMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SourceType, &m.SourceAttribute,
			&m.SourceValue, &m.GGIDRole, &m.GGIDAttribute, &m.Priority, &m.Enabled); err != nil {
			slog.Warn("attribute mapping scan error", "error", err)
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *attributeMappingRepo) Get(ctx context.Context, id string) (*AttributeMapping, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id::text, source_type, source_attribute, source_value,
		        ggid_role, ggid_attribute, priority, enabled
		 FROM attribute_mappings WHERE id = $1`, id)

	var m AttributeMapping
	err := row.Scan(&m.ID, &m.TenantID, &m.SourceType, &m.SourceAttribute,
		&m.SourceValue, &m.GGIDRole, &m.GGIDAttribute, &m.Priority, &m.Enabled)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *attributeMappingRepo) Create(ctx context.Context, m *AttributeMapping) error {
	if m.ID == "" {
		m.ID = "map-" + uuid.New().String()[:8]
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO attribute_mappings (id, tenant_id, source_type, source_attribute, source_value, ggid_role, ggid_attribute, priority, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		m.ID, m.TenantID, m.SourceType, m.SourceAttribute, m.SourceValue,
		m.GGIDRole, m.GGIDAttribute, m.Priority, m.Enabled)
	return err
}

func (r *attributeMappingRepo) Update(ctx context.Context, m *AttributeMapping) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attribute_mappings SET
		   source_type = $2, source_attribute = $3, source_value = $4,
		   ggid_role = $5, ggid_attribute = $6, priority = $7, enabled = $8
		 WHERE id = $1`,
		m.ID, m.SourceType, m.SourceAttribute, m.SourceValue,
		m.GGIDRole, m.GGIDAttribute, m.Priority, m.Enabled)
	return err
}

func (r *attributeMappingRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM attribute_mappings WHERE id = $1`, id)
	return err
}

// ResolveMappings applies attribute mappings to a set of legacy attributes
// and returns the GGID roles and custom attributes to assign.
func (r *attributeMappingRepo) ResolveMappings(ctx context.Context, tenantID uuid.UUID, legacyAttrs map[string][]string) (roles []string, attrs map[string]string) {
	mappings, err := r.List(ctx, tenantID)
	if err != nil {
		return nil, nil
	}

	seen := make(map[string]bool)
	attrs = make(map[string]string)

	for _, m := range mappings {
		if !m.Enabled {
			continue
		}
		values, ok := legacyAttrs[m.SourceAttribute]
		if !ok {
			continue
		}
		for _, v := range values {
			// Match: exact match or wildcard suffix match (e.g. "CN=Admins*").
			if matchValue(m.SourceValue, v) {
				if m.GGIDRole != "" && !seen[m.GGIDRole] {
					roles = append(roles, m.GGIDRole)
					seen[m.GGIDRole] = true
				}
				if m.GGIDAttribute != "" {
					attrs[m.GGIDAttribute] = v
				}
			}
		}
	}
	return roles, attrs
}

// matchValue checks if the source value matches (exact or wildcard suffix).
func matchValue(pattern, value string) bool {
	if pattern == value {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return false
}

// TestMappingResult is the response from the test endpoint.
type TestMappingResult struct {
	InputAttributes  map[string][]string `json:"input_attributes"`
	ResolvedRoles    []string            `json:"resolved_roles"`
	ResolvedAttrs    map[string]string   `json:"resolved_attributes"`
	MatchedMappings  int                 `json:"matched_mappings"`
}

// TestMapping simulates applying mappings to a sample input.
func (r *attributeMappingRepo) TestMapping(ctx context.Context, tenantID uuid.UUID, input map[string][]string) (*TestMappingResult, error) {
	roles, attrs := r.ResolveMappings(ctx, tenantID, input)
	matched := 0
	if len(roles) > 0 || len(attrs) > 0 {
		matched = len(roles) + len(attrs)
	}
	return &TestMappingResult{
		InputAttributes: input,
		ResolvedRoles:   roles,
		ResolvedAttrs:   attrs,
		MatchedMappings: matched,
	}, nil
}

// ValidateMapping checks that a mapping has required fields.
func ValidateMapping(m *AttributeMapping) error {
	if m.SourceAttribute == "" {
		return fmt.Errorf("source_attribute is required")
	}
	if m.SourceValue == "" {
		return fmt.Errorf("source_value is required")
	}
	if m.GGIDRole == "" && m.GGIDAttribute == "" {
		return fmt.Errorf("at least one of ggid_role or ggid_attribute is required")
	}
	return nil
}

// Ensure json is imported.
var _ = json.Marshal
