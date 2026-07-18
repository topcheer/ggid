package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func newTestHandlerWithAAGUIDRepo() *Handler {
	return &Handler{
		aaguidAllowlistRepo: repository.NewAAGUIDAllowlistRepository(nil),
	}
}

// Test 1: GET returns empty list with nil pool.
func TestAAGUIDAllowlist_ListEmpty(t *testing.T) {
	h := newTestHandlerWithAAGUIDRepo()

	req := httptest.NewRequest("GET", "/api/v1/auth/webauthn/aaguid", nil)
	w := httptest.NewRecorder()

	h.handleAAGUIDAllowlist(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 2: POST with valid AAGUID returns 201.
func TestAAGUIDAllowlist_Add(t *testing.T) {
	h := newTestHandlerWithAAGUIDRepo()

	body := `{"aaguid":"cb69481e-8ff7-4039-93ec-0a2729a154a8","name":"YubiKey 5 NFC","description":"Yubico"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/webauthn/aaguid", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleAAGUIDAllowlist(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var rec repository.AAGUIDRecord
	if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if rec.Name != "YubiKey 5 NFC" {
		t.Errorf("expected name=YubiKey 5 NFC, got %s", rec.Name)
	}
	if rec.Status != repository.AAGUIDStatusApproved {
		t.Errorf("expected status=approved, got %s", rec.Status)
	}
}

// Test 3: POST without aaguid returns 400.
func TestAAGUIDAllowlist_AddValidation(t *testing.T) {
	h := newTestHandlerWithAAGUIDRepo()

	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/webauthn/aaguid", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleAAGUIDAllowlist(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: DELETE with AAGUID returns 200.
func TestAAGUIDAllowlist_Remove(t *testing.T) {
	h := newTestHandlerWithAAGUIDRepo()

	req := httptest.NewRequest("DELETE", "/api/v1/auth/webauthn/aaguid/cb69481e-8ff7-4039-93ec-0a2729a154a8", nil)
	w := httptest.NewRecorder()

	h.handleAAGUIDAllowlist(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: IsApproved returns true with nil pool (allow all).
func TestAAGUIDAllowlist_IsApprovedNilPool(t *testing.T) {
	repo := repository.NewAAGUIDAllowlistRepository(nil)

	// With nil pool, any AAGUID should be allowed.
	if !repo.IsApproved(context.Background(), "some-aaguid") {
		t.Error("with nil pool, IsApproved should return true (allow all)")
	}
}

// Test 6: CheckAttestation returns nil with nil pool.
func TestAAGUIDAllowlist_CheckAttestationNilPool(t *testing.T) {
	repo := repository.NewAAGUIDAllowlistRepository(nil)

	err := repo.CheckAttestation(context.Background(), "test-aaguid")
	if err != nil {
		t.Errorf("with nil pool, CheckAttestation should return nil, got %v", err)
	}
}

// Test 7: SeedDefaults runs without error with nil pool.
func TestAAGUIDAllowlist_SeedDefaults(t *testing.T) {
	repo := repository.NewAAGUIDAllowlistRepository(nil)

	err := repo.SeedDefaults(context.Background(), "admin")
	if err != nil {
		t.Errorf("SeedDefaults with nil pool should not error: %v", err)
	}
}

// Test 8: CheckAAGUIDDuringRegistration returns true with nil repo.
func TestAAGUIDAllowlist_RegistrationCheckNilRepo(t *testing.T) {
	h := &Handler{} // no aaguidAllowlistRepo set

	req := httptest.NewRequest("POST", "/api/v1/auth/passkey/register/finish", nil)
	if !h.CheckAAGUIDDuringRegistration(req, "any-aaguid") {
		t.Error("with nil repo, CheckAAGUIDDuringRegistration should return true")
	}
}

// Test 9: GetByID returns nil with nil pool.
func TestAAGUIDAllowlist_GetByIDNilPool(t *testing.T) {
	repo := repository.NewAAGUIDAllowlistRepository(nil)

	rec, err := repo.GetByID(context.Background(), uuid.New().String())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if rec != nil {
		t.Error("with nil pool, GetByID should return nil")
	}
}
