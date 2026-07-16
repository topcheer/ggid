package handler

import (
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/google/uuid"
)

func TestParseUUID_Valid(t *testing.T) {
	id := uuid.New()
	got, err := parseUUID(id.String(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Errorf("expected %s, got %s", id, got)
	}
}

func TestParseUUID_Invalid(t *testing.T) {
	_, err := parseUUID("not-a-uuid", "test_field")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestParseOptionalUUID_Empty(t *testing.T) {
	got, err := parseOptionalUUID("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty string, got %v", got)
	}
}

func TestParseOptionalUUID_Valid(t *testing.T) {
	id := uuid.New()
	got, err := parseOptionalUUID(id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || *got != id {
		t.Errorf("expected %s, got %v", id, got)
	}
}

func TestParseOptionalUUID_Invalid(t *testing.T) {
	_, err := parseOptionalUUID("bad-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestJsonToMap_Empty(t *testing.T) {
	m := jsonToMap("")
	if m != nil {
		t.Errorf("expected nil for empty string, got %v", m)
	}
}

func TestJsonToMap_Valid(t *testing.T) {
	m := jsonToMap(`{"key":"value","num":123}`)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
}

func TestJsonToMap_Invalid(t *testing.T) {
	m := jsonToMap("not json")
	if m != nil {
		t.Errorf("expected nil for invalid JSON, got %v", m)
	}
}

func TestToGRPCError_NotFound(t *testing.T) {
	err := errors.New(errors.ErrNotFound, "not found")
	grpcErr := toGRPCError(err)
	if grpcErr == nil {
		t.Fatal("expected error")
	}
}

func TestToGRPCError_AlreadyExists(t *testing.T) {
	err := errors.New(errors.ErrAlreadyExists, "exists")
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

func TestToGRPCError_GenericError(t *testing.T) {
	grpcErr := toGRPCError(errors.New(errors.ErrInternal, "generic"))
	if grpcErr == nil {
		t.Fatal("expected error for generic error")
	}
}

func TestTenantToProto(t *testing.T) {
	now := time.Now().UTC()
	tenant := &domain.Tenant{
		ID:     uuid.New(),
		Name:   "Acme Corp",
		Slug:   "acme",
		Plan:   domain.PlanEnterprise,
		Status: domain.TenantActive,
		MaxUsers: 100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p := tenantToProto(tenant)
	if p.Name != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %s", p.Name)
	}
	if p.Slug != "acme" {
		t.Errorf("expected acme, got %s", p.Slug)
	}
	if p.MaxUsers != 100 {
		t.Errorf("expected 100, got %d", p.MaxUsers)
	}
	if p.CreatedAt == nil {
		t.Error("expected non-nil CreatedAt")
	}
}

func TestTenantToProto_ZeroTime(t *testing.T) {
	tenant := &domain.Tenant{
		ID:   uuid.New(),
		Name: "Test",
	}
	p := tenantToProto(tenant)
	if p.CreatedAt != nil {
		t.Error("zero CreatedAt should produce nil timestamp")
	}
}

func TestOrgToProto(t *testing.T) {
	parentID := uuid.New()
	org := &domain.Organization{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		ParentID: &parentID,
		Name:     "Engineering",
		Path:     "root.engineering",
	}
	p := orgToProto(org)
	if p.Name != "Engineering" {
		t.Errorf("expected Engineering, got %s", p.Name)
	}
	if p.ParentId == nil || *p.ParentId != parentID.String() {
		t.Error("parent ID mismatch")
	}
}

func TestOrgToProto_NoParent(t *testing.T) {
	org := &domain.Organization{
		ID:   uuid.New(),
		Name: "Root Org",
	}
	p := orgToProto(org)
	if p.ParentId != nil {
		t.Error("nil parent should produce nil proto field")
	}
}

func TestDeptToProto(t *testing.T) {
	dept := &domain.Department{
		ID:   uuid.New(),
		Name: "Platform Team",
		Path: "root.eng.platform",
	}
	p := deptToProto(dept)
	if p.Name != "Platform Team" {
		t.Errorf("expected Platform Team, got %s", p.Name)
	}
}

func TestTeamToProto(t *testing.T) {
	team := &domain.Team{
		ID:          uuid.New(),
		Name:        "SRE",
		Description: "Site Reliability",
	}
	p := teamToProto(team)
	if p.Name != "SRE" {
		t.Errorf("expected SRE, got %s", p.Name)
	}
	if p.Description != "Site Reliability" {
		t.Errorf("expected Site Reliability, got %s", p.Description)
	}
}
