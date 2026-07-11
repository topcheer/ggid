package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPermissionInheritance_SingleNode(t *testing.T) {
	tree := NewPermissionTree()
	id := uuid.New()
	tree.AddNode(id, uuid.Nil, []string{"read", "write"})

	perms, err := tree.GetEffectivePermissions(context.Background(), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(perms) != 2 {
		t.Errorf("expected 2 perms, got %d", len(perms))
	}
}

func TestPermissionInheritance_ParentInherit(t *testing.T) {
	tree := NewPermissionTree()
	parent := uuid.New()
	child := uuid.New()
	tree.AddNode(parent, uuid.Nil, []string{"admin:read"})
	tree.AddNode(child, parent, []string{"docs:write"})

	perms, _ := tree.GetEffectivePermissions(context.Background(), child)
	if len(perms) != 2 {
		t.Errorf("child should inherit parent perms, got %d", len(perms))
	}
	if !tree.HasPermission(context.Background(), child, "admin:read") {
		t.Error("child should have inherited admin:read")
	}
}

func TestPermissionInheritance_DeepChain(t *testing.T) {
	tree := NewPermissionTree()
	root := uuid.New()
	mid := uuid.New()
	leaf := uuid.New()
	tree.AddNode(root, uuid.Nil, []string{"root:perm"})
	tree.AddNode(mid, root, []string{"mid:perm"})
	tree.AddNode(leaf, mid, []string{"leaf:perm"})

	perms, _ := tree.GetEffectivePermissions(context.Background(), leaf)
	if len(perms) != 3 {
		t.Errorf("leaf should inherit 3 perms, got %d", len(perms))
	}
}

func TestPermissionInheritance_Deduplicates(t *testing.T) {
	tree := NewPermissionTree()
	parent := uuid.New()
	child := uuid.New()
	tree.AddNode(parent, uuid.Nil, []string{"read", "write"})
	tree.AddNode(child, parent, []string{"read", "delete"})

	perms, _ := tree.GetEffectivePermissions(context.Background(), child)
	count := 0
	for _, p := range perms {
		if p == "read" {
			count++
		}
	}
	if count != 1 {
		t.Error("read should appear only once after dedup")
	}
}

func TestPermissionInheritance_CycleDetection(t *testing.T) {
	tree := NewPermissionTree()
	a, b := uuid.New(), uuid.New()
	tree.AddNode(a, b, []string{"x"})
	tree.AddNode(b, a, []string{"y"})

	_, err := tree.GetEffectivePermissions(context.Background(), a)
	if err == nil {
		t.Error("should detect cycle")
	}
}

func TestPermissionInheritance_TreeDepth(t *testing.T) {
	tree := NewPermissionTree()
	root, mid, leaf := uuid.New(), uuid.New(), uuid.New()
	tree.AddNode(root, uuid.Nil, nil)
	tree.AddNode(mid, root, nil)
	tree.AddNode(leaf, mid, nil)

	if tree.GetTreeDepth(leaf) != 3 {
		t.Errorf("expected depth 3, got %d", tree.GetTreeDepth(leaf))
	}
}

func TestPermissionInheritance_Wildcard(t *testing.T) {
	tree := NewPermissionTree()
	parent := uuid.New()
	child := uuid.New()
	tree.AddNode(parent, uuid.Nil, []string{"*"})
	tree.AddNode(child, parent, nil)

	if !tree.HasPermission(context.Background(), child, "anything") {
		t.Error("wildcard should grant all to children")
	}
}

func TestPermissionInheritance_NonExistentNode(t *testing.T) {
	tree := NewPermissionTree()
	perms, err := tree.GetEffectivePermissions(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("should not error: %v", err)
	}
	if len(perms) != 0 {
		t.Error("missing node should have no permissions")
	}
}

var _ = sync.Mutex{}
var _ = time.Now
