package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateTrustChain_Revoked(t *testing.T) {
	e := &FederationEntity{TrustLevel: "revoked", Enabled: true}
	if err := ValidateTrustChain(e, ""); err == nil {
		t.Error("revoked entity should fail validation")
	}
}

func TestValidateTrustChain_Expired(t *testing.T) {
	exp := time.Now().Add(-1 * time.Hour)
	e := &FederationEntity{TrustLevel: "verified", Enabled: true, ExpiresAt: &exp}
	if err := ValidateTrustChain(e, ""); err == nil {
		t.Error("expired entity should fail validation")
	}
}

func TestValidateTrustChain_Valid(t *testing.T) {
	e := &FederationEntity{TrustLevel: "verified", Enabled: true}
	if err := ValidateTrustChain(e, ""); err != nil {
		t.Errorf("valid entity should pass: %v", err)
	}
}

func TestTransformAssertion_RenameAndFilter(t *testing.T) {
	claims := map[string]any{"givenName": "Alice", "email": "alice@example.com", "ssn": "123-456"}
	rule := &TransformRule{
		ClaimMappings: map[string]any{"first_name": "givenName"},
		ClaimFilters:  []string{"ssn"},
	}
	result := TransformAssertion(claims, rule)
	if result["first_name"] != "Alice" {
		t.Error("should rename givenName to first_name")
	}
	if _, exists := result["givenName"]; exists {
		t.Error("source key should be removed after rename")
	}
	if _, exists := result["ssn"]; exists {
		t.Error("ssn should be filtered")
	}
}

func TestCertExpiringSoon(t *testing.T) {
	soon := time.Now().Add(15 * 24 * time.Hour)
	far := time.Now().Add(365 * 24 * time.Hour)
	entity := &FederationEntity{
		Certificates: []FedCert{{KID: "k1", ExpiresAt: soon}},
		ExpiresAt:    &far,
	}
	if !CertExpiringSoon(entity, 30) {
		t.Error("cert expiring in 15d should warn within 30d")
	}
	if CertExpiringSoon(entity, 10) {
		t.Error("cert expiring in 15d should not warn within 10d")
	}
}

func TestCertFingerprint_Deterministic(t *testing.T) {
	fp1 := CertFingerprint("test-cert-pem")
	fp2 := CertFingerprint("test-cert-pem")
	if fp1 != fp2 {
		t.Error("fingerprint should be deterministic")
	}
	if CertFingerprint("different") == fp1 {
		t.Error("different PEM should have different fingerprint")
	}
}

func TestFederationRepo_NilPool(t *testing.T) {
	repo := newFederationRepo(nil)
	entities, err := repo.ListEntities(nil, uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(entities) != 0 {
		t.Error("nil pool should return empty list")
	}
}
