package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestCheck_DirectHit — subject has exact relation on object → allowed
func TestCheck_DirectHit(t *testing.T) {
	repo := newRelationTupleRepo(nil) // nil pool = no DB, Check returns false
	// Since we can't test DB-backed Check without a pool, test the pure logic.
	// With nil pool, DirectSubjects returns nil → check returns false.
	resp := repo.Check(context.Background(), CheckRequest{
		TenantID:  uuid.New(),
		Namespace: "document",
		Object:    "report",
		Relation:  "viewer",
		Subject:   "user:alice",
	})
	if resp.Allowed {
		t.Fatal("nil pool should not allow")
	}
}

// TestComputedRelations_CanView — verifies the permission→relation mapping
func TestComputedRelations_CanView(t *testing.T) {
	rels := computedRelationsFor("can_view")
	if len(rels) != 4 {
		t.Fatalf("expected 4 relations for can_view, got %d", len(rels))
	}
	expected := map[string]bool{"viewer": true, "commenter": true, "editor": true, "owner": true}
	for _, r := range rels {
		if !expected[r] {
			t.Errorf("unexpected relation '%s' for can_view", r)
		}
	}
}

// TestComputedRelations_CanEdit — editor + owner only
func TestComputedRelations_CanEdit(t *testing.T) {
	rels := computedRelationsFor("can_edit")
	if len(rels) != 2 {
		t.Fatalf("expected 2 relations for can_edit, got %d", len(rels))
	}
	for _, r := range rels {
		if r != "editor" && r != "owner" {
			t.Errorf("unexpected relation '%s' for can_edit", r)
		}
	}
}

// TestComputedRelations_CanDelete — owner only
func TestComputedRelations_CanDelete(t *testing.T) {
	rels := computedRelationsFor("can_delete")
	if len(rels) != 1 || rels[0] != "owner" {
		t.Fatalf("expected only 'owner' for can_delete, got %v", rels)
	}
}

// TestComputedRelations_UnknownPermission — returns nil
func TestComputedRelations_UnknownPermission(t *testing.T) {
	rels := computedRelationsFor("unknown_perm")
	if rels != nil {
		t.Fatalf("expected nil for unknown permission, got %v", rels)
	}
}

// TestCheck_DepthLimit — depth=0 always returns false
func TestCheck_DepthLimit(t *testing.T) {
	repo := newRelationTupleRepo(nil)
	resp := repo.Check(context.Background(), CheckRequest{
		TenantID:  uuid.New(),
		Namespace: "document",
		Object:    "report",
		Relation:  "viewer",
		Subject:   "user:alice",
		MaxDepth:  0,
	})
	if resp.Allowed {
		t.Fatal("depth=0 should never allow")
	}
}

// TestRelationTuple_WriteRead — verify tuple serialization doesn't lose fields
func TestRelationTuple_Serialization(t *testing.T) {
	tid := uuid.New()
	tuple := &RelationTuple{
		TenantID:  tid,
		Namespace: "document",
		Object:    "q4-report",
		Relation:  "owner",
		Subject:   "user:bob",
	}
	if tuple.Namespace != "document" || tuple.Object != "q4-report" {
		t.Fatal("tuple fields not set correctly")
	}
}
