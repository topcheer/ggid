package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RelationTuple represents a Zanzibar-style relationship tuple: (object, relation, subject).
// Format follows OpenFGA: namespace:object_id, e.g. "document:report-q4", "user:alice".
type RelationTuple struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Namespace string    `json:"namespace"` // e.g. "document", "folder", "user"
	Object    string    `json:"object"`    // e.g. "report-q4" (without namespace prefix)
	Relation  string    `json:"relation"`  // e.g. "owner", "viewer", "editor", "parent"
	Subject   string    `json:"subject"`   // e.g. "user:alice" or "group:finance"
	CreatedAt time.Time `json:"created_at"`
}

// CheckRequest is the input for a ReBAC permission check.
type CheckRequest struct {
	TenantID   uuid.UUID
	Namespace  string // e.g. "document"
	Object     string // e.g. "report-q4"
	Relation   string // e.g. "can_view" (permission) or "viewer" (direct relation)
	Subject    string // e.g. "user:alice"
	MaxDepth   int    // traversal depth limit (default 25)
}

// CheckResponse is the result of a ReBAC permission check.
type CheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// relationTupleRepo manages relation_tuples persistence.
type relationTupleRepo struct {
	pool *pgxpool.Pool
}

func newRelationTupleRepo(pool *pgxpool.Pool) *relationTupleRepo {
	return &relationTupleRepo{pool: pool}
}

// EnsureSchema creates the relation_tuples table.
func (r *relationTupleRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS relation_tuples (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id  UUID NOT NULL,
			namespace  TEXT NOT NULL,
			object     TEXT NOT NULL,
			relation   TEXT NOT NULL,
			subject    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (tenant_id, namespace, object, relation, subject)
		);
		CREATE INDEX IF NOT EXISTS idx_rt_check ON relation_tuples (tenant_id, namespace, object, relation);
		CREATE INDEX IF NOT EXISTS idx_rt_subject ON relation_tuples (tenant_id, subject);
	`)
	return err
}

// WriteTuple stores a relationship tuple.
func (r *relationTupleRepo) WriteTuple(ctx context.Context, t *RelationTuple) error {
	if r.pool == nil {
		return nil
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO relation_tuples (id, tenant_id, namespace, object, relation, subject)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, namespace, object, relation, subject) DO NOTHING`,
		t.ID, t.TenantID, t.Namespace, t.Object, t.Relation, t.Subject,
	)
	return err
}

// DeleteTuple removes a relationship tuple.
func (r *relationTupleRepo) DeleteTuple(ctx context.Context, tenantID uuid.UUID, ns, obj, rel, subj string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		DELETE FROM relation_tuples
		WHERE tenant_id = $1 AND namespace = $2 AND object = $3 AND relation = $4 AND subject = $5`,
		tenantID, ns, obj, rel, subj,
	)
	return err
}

// ReadTuples returns tuples matching the given filters. Empty values = wildcard.
func (r *relationTupleRepo) ReadTuples(ctx context.Context, tenantID uuid.UUID, ns, obj, rel, subj string) ([]*RelationTuple, error) {
	if r.pool == nil {
		return []*RelationTuple{}, nil
	}
	query := `SELECT id, tenant_id, namespace, object, relation, subject, created_at FROM relation_tuples WHERE tenant_id = $1`
	args := []any{tenantID}
	idx := 2
	if ns != "" {
		query += fmt.Sprintf(" AND namespace = $%d", idx); args = append(args, ns); idx++
	}
	if obj != "" {
		query += fmt.Sprintf(" AND object = $%d", idx); args = append(args, obj); idx++
	}
	if rel != "" {
		query += fmt.Sprintf(" AND relation = $%d", idx); args = append(args, rel); idx++
	}
	if subj != "" {
		query += fmt.Sprintf(" AND subject = $%d", idx); args = append(args, subj); idx++
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*RelationTuple
	for rows.Next() {
		var t RelationTuple
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Namespace, &t.Object, &t.Relation, &t.Subject, &t.CreatedAt); err != nil {
			continue
		}
		result = append(result, &t)
	}
	return result, nil
}

// DirectSubjects returns subjects that have a direct relation to an object.
// Used by the Check engine for graph traversal.
func (r *relationTupleRepo) DirectSubjects(ctx context.Context, tenantID uuid.UUID, ns, obj, rel string) ([]string, error) {
	if r.pool == nil {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT subject FROM relation_tuples
		WHERE tenant_id = $1 AND namespace = $2 AND object = $3 AND relation = $4`,
		tenantID, ns, obj, rel,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			continue
		}
		subjects = append(subjects, s)
	}
	return subjects, nil
}

// Check performs a ReBAC permission check via recursive graph traversal.
// Resolution order:
//  1. Direct: subject has the exact relation on the object
//  2. Computed: relation is a permission that maps to other relations (e.g. can_view → viewer|commenter)
//  3. Transitive: subject is reached via nested objects (e.g. folder viewer → document viewer via parent)
func (r *relationTupleRepo) Check(ctx context.Context, req CheckRequest) CheckResponse {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 25
	}
	if r.pool == nil {
		return CheckResponse{Allowed: false, Reason: "ReBAC not configured (no DB)"}
	}

	visited := make(map[string]bool)
	allowed := r.checkRecursive(ctx, req, req.Namespace, req.Object, req.Relation, req.Subject, req.MaxDepth, visited)
	if allowed {
		return CheckResponse{Allowed: true}
	}
	return CheckResponse{Allowed: false, Reason: "no matching relationship found"}
}

// checkRecursive traverses the relationship graph.
func (r *relationTupleRepo) checkRecursive(ctx context.Context, req CheckRequest, ns, obj, rel, subject string, depth int, visited map[string]bool) bool {
	if depth <= 0 {
		return false
	}
	key := ns + ":" + obj + "#" + rel + "@" + subject
	if visited[key] {
		return false
	}
	visited[key] = true

	// 1. Direct match: does subject have this exact relation on this object?
	direct, err := r.DirectSubjects(ctx, req.TenantID, ns, obj, rel)
	if err == nil {
		for _, s := range direct {
			if s == subject {
				return true
			}
			// Subject groups: if subject is "group:engineering" and we're checking "user:alice",
			// check if alice is a member of the group.
			if strings.HasPrefix(s, "group:") {
				groupName := strings.TrimPrefix(s, "group:")
				if r.checkRecursive(ctx, req, "group", groupName, "member", subject, depth-1, visited) {
					return true
				}
			}
		}
	}

	// 2. Computed permissions: common mappings.
	// can_view → viewer, commenter, editor, owner
	// can_edit → editor, owner
	// can_delete → owner
	computedRelations := computedRelationsFor(rel)
	for _, altRel := range computedRelations {
		if r.checkRecursive(ctx, req, ns, obj, altRel, subject, depth-1, visited) {
			return true
		}
	}

	// 3. Transitive via parent: check if parent objects grant the relation.
	parents, err := r.DirectSubjects(ctx, req.TenantID, ns, obj, "parent")
	if err == nil {
		for _, parent := range parents {
			parentParts := strings.SplitN(parent, ":", 2)
			if len(parentParts) != 2 {
				continue
			}
			parentNS, parentObj := parentParts[0], parentParts[1]
			// If subject has the relation on the parent, they inherit it on the child.
			if r.checkRecursive(ctx, req, parentNS, parentObj, rel, subject, depth-1, visited) {
				return true
			}
		}
	}

	return false
}

// computedRelationsFor returns the set of direct relations that satisfy a computed permission.
func computedRelationsFor(permission string) []string {
	switch permission {
	case "can_view":
		return []string{"viewer", "commenter", "editor", "owner"}
	case "can_edit":
		return []string{"editor", "owner"}
	case "can_delete":
		return []string{"owner"}
	case "can_comment":
		return []string{"commenter", "editor", "owner"}
	default:
		return nil
	}
}
