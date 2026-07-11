package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDeptRBAC_AssignAndCheck(t *testing.T) {
	ResetDeptRoleStore()
	deptID := uuid.New()
	userID := uuid.New()

	_, err := AssignDeptRole(context.Background(), deptID, userID, "dept.admin", []string{"read", "write"})
	if err != nil {
		t.Fatalf("AssignDeptRole: %v", err)
	}

	if !CheckDeptPermission(context.Background(), deptID, userID, "read") {
		t.Error("should have read permission")
	}
	if !CheckDeptPermission(context.Background(), deptID, userID, "write") {
		t.Error("should have write permission")
	}
}

func TestDeptRBAC_NoPermission(t *testing.T) {
	ResetDeptRoleStore()
	deptID := uuid.New()
	userID := uuid.New()
	AssignDeptRole(context.Background(), deptID, userID, "dept.member", []string{"read"})

	if CheckDeptPermission(context.Background(), deptID, userID, "delete") {
		t.Error("should NOT have delete permission")
	}
}

func TestDeptRBAC_DifferentDept(t *testing.T) {
	ResetDeptRoleStore()
	dept1 := uuid.New()
	dept2 := uuid.New()
	userID := uuid.New()

	AssignDeptRole(context.Background(), dept1, userID, "dept.admin", []string{"read"})

	if CheckDeptPermission(context.Background(), dept2, userID, "read") {
		t.Error("should NOT have permission in different department")
	}
}

func TestDeptRBAC_WildcardPermission(t *testing.T) {
	ResetDeptRoleStore()
	deptID := uuid.New()
	userID := uuid.New()

	AssignDeptRole(context.Background(), deptID, userID, "dept.super", []string{"*"})

	if !CheckDeptPermission(context.Background(), deptID, userID, "anything") {
		t.Error("wildcard should grant all permissions")
	}
}

func TestDeptRBAC_ListRolesForUser(t *testing.T) {
	ResetDeptRoleStore()
	userID := uuid.New()
	AssignDeptRole(context.Background(), uuid.New(), userID, "dept.admin", []string{"read"})
	AssignDeptRole(context.Background(), uuid.New(), userID, "dept.member", []string{"read"})

	roles, _ := ListDeptRoles(context.Background(), userID)
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestDeptRBAC_ListMembersWithRole(t *testing.T) {
	ResetDeptRoleStore()
	deptID := uuid.New()
	u1, u2 := uuid.New(), uuid.New()

	AssignDeptRole(context.Background(), deptID, u1, "dept.admin", []string{"read"})
	AssignDeptRole(context.Background(), deptID, u2, "dept.admin", []string{"read"})
	AssignDeptRole(context.Background(), deptID, uuid.New(), "dept.member", []string{"read"})

	users, _ := ListDeptMembersWithRole(context.Background(), deptID, "dept.admin")
	if len(users) != 2 {
		t.Errorf("expected 2 admins, got %d", len(users))
	}
}

func TestDeptRBAC_InvalidAssign(t *testing.T) {
	ResetDeptRoleStore()
	_, err := AssignDeptRole(context.Background(), uuid.Nil, uuid.New(), "role", []string{"r"})
	if err == nil {
		t.Error("should reject nil dept_id")
	}
	_, err = AssignDeptRole(context.Background(), uuid.New(), uuid.New(), "", []string{"r"})
	if err == nil {
		t.Error("should reject empty role")
	}
}
