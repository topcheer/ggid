package idpconfig

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func newTestSvc() *Service {
	return NewService(NewMemoryStore())
}

// 1. TestCreate_Success
func TestCreate_Success(t *testing.T) {
	svc := newTestSvc()
	tenantID := uuid.New()
	cfg, err := svc.Create(context.Background(), tenantID, IdPTypeSAML, "Okta SAML", `{"entity_id":"https://okta.com/saml"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if cfg.IdPType != IdPTypeSAML {
		t.Fatalf("expected saml, got %s", cfg.IdPType)
	}
	if !cfg.Enabled {
		t.Fatal("expected enabled by default")
	}
}

// 2. TestCreate_InvalidType
func TestCreate_InvalidType(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.Create(context.Background(), uuid.New(), "invalid", "test", "{}")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// 3. TestCreate_MissingFields
func TestCreate_MissingFields(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.Create(context.Background(), uuid.Nil, IdPTypeOIDC, "test", "{}")
	if err == nil {
		t.Fatal("expected error for nil tenant")
	}
	_, err = svc.Create(context.Background(), uuid.New(), IdPTypeOIDC, "", "{}")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// 4. TestGet_NotFound
func TestGet_NotFound(t *testing.T) {
	svc := newTestSvc()
	_, err := svc.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// 5. TestList_ByTenant
func TestList_ByTenant(t *testing.T) {
	svc := newTestSvc()
	t1 := uuid.New()
	t2 := uuid.New()
	svc.Create(context.Background(), t1, IdPTypeSAML, "SAML1", "{}")
	svc.Create(context.Background(), t1, IdPTypeOIDC, "OIDC1", "{}")
	svc.Create(context.Background(), t2, IdPTypeLDAP, "LDAP1", "{}")

	t1Configs, _ := svc.List(context.Background(), t1)
	if len(t1Configs) != 2 {
		t.Fatalf("expected 2 for t1, got %d", len(t1Configs))
	}
	t2Configs, _ := svc.List(context.Background(), t2)
	if len(t2Configs) != 1 {
		t.Fatalf("expected 1 for t2, got %d", len(t2Configs))
	}
}

// 6. TestUpdate_Success
func TestUpdate_Success(t *testing.T) {
	svc := newTestSvc()
	cfg, _ := svc.Create(context.Background(), uuid.New(), IdPTypeSAML, "Original", "{}")
	updated, err := svc.Update(context.Background(), cfg.ID, "Renamed", `{"updated":true}`, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Fatal("name not updated")
	}
	if updated.Enabled {
		t.Fatal("expected disabled")
	}
}

// 7. TestDelete_Success
func TestDelete_Success(t *testing.T) {
	svc := newTestSvc()
	cfg, _ := svc.Create(context.Background(), uuid.New(), IdPTypeOIDC, "Test", "{}")
	err := svc.Delete(context.Background(), cfg.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.Get(context.Background(), cfg.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
