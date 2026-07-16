package handler

import (
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

func TestToGRPCError_NotFound(t *testing.T) {
	err := errors.New(errors.ErrNotFound, "role not found")
	grpcErr := toGRPCError(err)
	if grpcErr == nil {
		t.Fatal("expected error")
	}
}

func TestToGRPCError_AlreadyExists(t *testing.T) {
	err := errors.New(errors.ErrAlreadyExists, "role exists")
	grpcErr := toGRPCError(err)
	if grpcErr == nil {
		t.Fatal("expected error")
	}
}

func TestToGRPCError_InvalidArgument(t *testing.T) {
	err := errors.New(errors.ErrInvalidArgument, "bad arg")
	grpcErr := toGRPCError(err)
	if grpcErr == nil {
		t.Fatal("expected error")
	}
}

func TestToGRPCError_PermissionDenied(t *testing.T) {
	err := errors.New(errors.ErrPermissionDenied, "denied")
	grpcErr := toGRPCError(err)
	if grpcErr == nil {
		t.Fatal("expected error")
	}
}

func TestToGRPCError_GenericError(t *testing.T) {
	grpcErr := toGRPCError(errors.New(errors.ErrInternal, "something went wrong"))
	if grpcErr == nil {
		t.Fatal("expected error for generic error")
	}
}

func TestRoleToProto_Basic(t *testing.T) {
	now := time.Now().UTC()
	parentID := uuid.New()
	role := &domain.Role{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Key:          "admin",
		Name:         "Administrator",
		Description:  "Full access",
		SystemRole:   true,
		ParentRoleID: &parentID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	p := roleToProto(role)
	if p.Key != "admin" {
		t.Errorf("expected key admin, got %s", p.Key)
	}
	if p.Name != "Administrator" {
		t.Errorf("expected name Administrator, got %s", p.Name)
	}
	if !p.SystemRole {
		t.Error("expected system role")
	}
	if p.ParentRoleId == nil || *p.ParentRoleId != parentID.String() {
		t.Error("parent role ID mismatch")
	}
}

func TestRoleToProto_NoParent(t *testing.T) {
	role := &domain.Role{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Key:      "viewer",
		Name:     "Viewer",
	}

	p := roleToProto(role)
	if p.ParentRoleId != nil {
		t.Error("nil parent should produce nil proto field")
	}
	if p.CreatedAt != nil {
		t.Error("zero created_at should produce nil timestamp")
	}
}

func TestEmitAudit_NilPublisher(t *testing.T) {
	h := &RoleHandler{auditor: nil}
	h.emitAudit("test", "success", uuid.Nil, uuid.Nil, "roles", uuid.New().String())
}
