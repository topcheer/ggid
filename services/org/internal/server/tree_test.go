package httpserver

import (
	"testing"

	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

func TestBuildOrgTree(t *testing.T) {
	root := uuid.New()
	child := uuid.New()
	grandchild := uuid.New()
	orphan := uuid.New()
	pid := child
	orphParent := uuid.New() // parent not in result set
	orgs := []*domain.Organization{
		{ID: root, Name: "Root", Path: "root", ParentID: nil},
		{ID: child, Name: "Child", Path: "root/child", ParentID: &root},
		{ID: grandchild, Name: "Grandchild", Path: "root/child/grandchild", ParentID: &pid},
		{ID: orphan, Name: "Orphan", Path: "orphan", ParentID: &orphParent},
	}

	tree := buildOrgTree(orgs, 0)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
	if tree[0].ID != root.String() || tree[1].ID != orphan.String() {
		t.Fatalf("unexpected root order: %v, %v", tree[0].ID, tree[1].ID)
	}
	if len(tree[0].Children) != 1 || len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("expected nested children, got root children %d", len(tree[0].Children))
	}
}

func TestPruneTree(t *testing.T) {
	root := uuid.New()
	child := uuid.New()
	grandchild := uuid.New()
	pid := child
	orgs := []*domain.Organization{
		{ID: root, Name: "Root", Path: "root", ParentID: nil},
		{ID: child, Name: "Child", Path: "root/child", ParentID: &root},
		{ID: grandchild, Name: "Grandchild", Path: "root/child/grandchild", ParentID: &pid},
	}

	tree := buildOrgTree(orgs, 2)
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 child after prune, got %d", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 0 {
		t.Fatalf("expected grandchildren removed after prune, got %d", len(tree[0].Children[0].Children))
	}
}
