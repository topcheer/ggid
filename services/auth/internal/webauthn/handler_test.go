package webauthn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	wbn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Mock CredentialStore
// ---------------------------------------------------------------------------

type mockStore struct {
	creds        []*Credential
	saveErr      error
	getByUserErr error
	getByIDErr   error
	updateErr    error
	deleteErr    error
}

func (m *mockStore) SaveCredential(_ context.Context, cred *Credential) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.creds = append(m.creds, cred)
	return nil
}

func (m *mockStore) GetCredentialsByUser(_ context.Context, _, _ uuid.UUID) ([]*Credential, error) {
	if m.getByUserErr != nil {
		return nil, m.getByUserErr
	}
	return m.creds, nil
}

func (m *mockStore) GetCredentialByID(_ context.Context, _ uuid.UUID, credID []byte) (*Credential, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for _, c := range m.creds {
		if string(c.CredentialID) == string(credID) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("credential not found")
}

func (m *mockStore) UpdateCounter(_ context.Context, _ uuid.UUID, _ []byte, _ uint32) error {
	return m.updateErr
}

func (m *mockStore) UpdateLastUsed(_ context.Context, _ uuid.UUID, _ []byte, _ time.Time) error {
	return nil
}

func (m *mockStore) DeleteCredential(_ context.Context, _ uuid.UUID, _ []byte) error {
	return m.deleteErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const (
	testTenantIDStr = "00000000-0000-0000-0000-000000000001"
	testUserIDStr   = "00000000-0000-0000-0000-000000000002"
	testUserID2Str  = "00000000-0000-0000-0000-000000000003"
)

func testHandler(t *testing.T, store CredentialStore) *Handler {
	t.Helper()
	h, err := NewHandler("localhost", "Test RP", store)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func newReq(method, target string, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	return r
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Errorf("status = %d, want %d; body=%s", rr.Code, want, rr.Body.String())
	}
}

func assertBodyContains(t *testing.T, rr *httptest.ResponseRecorder, substr string) {
	t.Helper()
	if !strings.Contains(rr.Body.String(), substr) {
		t.Errorf("body does not contain %q; got %s", substr, rr.Body.String())
	}
}

// extractChallenge parses the begin-registration / begin-auth response JSON
// and returns the challenge string (base64url).
func extractChallenge(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		PublicKey struct {
			Challenge string `json:"challenge"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse challenge from response: %v; body=%s", err, body)
	}
	if resp.PublicKey.Challenge == "" {
		t.Fatalf("empty challenge in response body: %s", body)
	}
	return resp.PublicKey.Challenge
}

// buildRegBody constructs a CredentialCreationResponse JSON body with the
// given challenge. The attestation object uses "none" format with minimal
// authData that includes attested-credential-data (enough to pass Parse).
func buildRegBody(challenge string) string {
	cdJSON, _ := json.Marshal(map[string]string{
		"type":      "webauthn.create",
		"challenge": challenge,
		"origin":    "https://localhost",
	})
	cdB64 := base64.RawURLEncoding.EncodeToString(cdJSON)

	// authData: 37-byte header + 16 AAGUID + 2 credID-len + 1 pubKey byte = 56
	authData := make([]byte, 56)
	authData[32] = 0x41 // UP(0x01) + AT(0x40)

	attObj := map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": authData,
	}
	attBytes, _ := cbor.Marshal(attObj)
	attB64 := base64.RawURLEncoding.EncodeToString(attBytes)

	rawID := base64.RawURLEncoding.EncodeToString(make([]byte, 16))

	body := map[string]any{
		"id":    rawID,
		"type":  "public-key",
		"rawId": rawID,
		"response": map[string]any{
			"clientDataJSON":    cdB64,
			"attestationObject": attB64,
		},
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// buildAuthBody constructs a CredentialAssertionResponse JSON body with the
// given challenge. authenticatorData is 37 zero bytes (valid for Parse).
func buildAuthBody(challenge string) string {
	cdJSON, _ := json.Marshal(map[string]string{
		"type":      "webauthn.get",
		"challenge": challenge,
		"origin":    "https://localhost",
	})
	cdB64 := base64.RawURLEncoding.EncodeToString(cdJSON)

	authData := make([]byte, 37) // 37 zero bytes — valid for assertion Parse
	authDataB64 := base64.RawURLEncoding.EncodeToString(authData)

	sig := make([]byte, 64)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	rawID := base64.RawURLEncoding.EncodeToString(make([]byte, 16))

	body := map[string]any{
		"id":    rawID,
		"type":  "public-key",
		"rawId": rawID,
		"response": map[string]any{
			"clientDataJSON":    cdB64,
			"authenticatorData": authDataB64,
			"signature":         sigB64,
		},
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// ---------------------------------------------------------------------------
// NewHandler
// ---------------------------------------------------------------------------

func TestNewHandler_Success(t *testing.T) {
	h, err := NewHandler("localhost", "Test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("handler is nil")
	}
	if h.wbn == nil {
		t.Fatal("webauthn instance is nil")
	}
	if h.sessions == nil {
		t.Fatal("session store is nil")
	}
}

func TestNewHandler_EmptyRPID(t *testing.T) {
	// Empty RPID: webauthn.New may or may not reject it depending on version.
	// We just verify the function doesn't panic.
	_, _ = NewHandler("", "Test", nil)
}

func TestNewHandler_WithStore(t *testing.T) {
	store := &mockStore{}
	h, err := NewHandler("localhost", "Test", store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.creds == nil {
		t.Fatal("store not set")
	}
}

// ---------------------------------------------------------------------------
// RegisterRoutes
// ---------------------------------------------------------------------------

func TestRegisterRoutes(t *testing.T) {
	h := testHandler(t, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Verify routes are registered by making a request through the mux.
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials", "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	mux.ServeHTTP(rr, req)
	// Should NOT get 404 (route exists); will get 200 or other.
	if rr.Code == http.StatusNotFound {
		t.Fatal("route not registered")
	}
}

// ---------------------------------------------------------------------------
// Session Store
// ---------------------------------------------------------------------------

func TestSessionStore_SaveGetDelete(t *testing.T) {
	s := newSessionStore()

	sd := &sessionData{challenge: "abc"}
	s.save("reg:abc", sd)

	got, ok := s.get("reg:abc")
	if !ok {
		t.Fatal("expected session to exist")
	}
	if got.challenge != "abc" {
		t.Errorf("challenge = %q, want %q", got.challenge, "abc")
	}

	s.delete("reg:abc")
	if _, ok := s.get("reg:abc"); ok {
		t.Fatal("expected session to be deleted")
	}
}

func TestSessionStore_NotFound(t *testing.T) {
	s := newSessionStore()
	if _, ok := s.get("nonexistent"); ok {
		t.Fatal("expected not found")
	}
}

func TestSessionStore_Expiry(t *testing.T) {
	s := newSessionStore()
	sd := &sessionData{challenge: "expiring"}
	s.save("key", sd)

	// Manually backdate to simulate expiry.
	sd.createdAt = time.Now().Add(-10 * time.Minute)
	if _, ok := s.get("key"); ok {
		t.Fatal("expected expired session to be removed")
	}
}

// ---------------------------------------------------------------------------
// beginRegistration
// ---------------------------------------------------------------------------

func TestBeginRegistration_Errors(t *testing.T) {
	h := testHandler(t, nil)

	tests := []struct {
		name   string
		method string
		target string
		tenant string
		want   int
	}{
		{"wrong method", http.MethodGet, "/api/v1/webauthn/register/begin?user_id=" + testUserIDStr, testTenantIDStr, http.StatusMethodNotAllowed},
		{"missing tenant", http.MethodPost, "/api/v1/webauthn/register/begin?user_id=" + testUserIDStr, "", http.StatusBadRequest},
		{"invalid tenant", http.MethodPost, "/api/v1/webauthn/register/begin?user_id=" + testUserIDStr, "not-a-uuid", http.StatusBadRequest},
		{"missing user_id", http.MethodPost, "/api/v1/webauthn/register/begin", testTenantIDStr, http.StatusBadRequest},
		{"invalid user_id", http.MethodPost, "/api/v1/webauthn/register/begin?user_id=bad", testTenantIDStr, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := newReq(tc.method, tc.target, "")
			if tc.tenant != "" {
				req.Header.Set("X-Tenant-ID", tc.tenant)
			}
			h.beginRegistration(rr, req)
			assertStatus(t, rr, tc.want)
		})
	}
}

func TestBeginRegistration_Success(t *testing.T) {
	h := testHandler(t, nil)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodPost, "/api/v1/webauthn/register/begin?user_id="+testUserIDStr, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.beginRegistration(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "challenge")

	// Verify session was stored.
	ch := extractChallenge(t, rr.Body.Bytes())
	if _, ok := h.sessions.get("reg:" + ch); !ok {
		t.Fatal("session not stored after beginRegistration")
	}
}

func TestBeginRegistration_WithStore(t *testing.T) {
	// Verify buildWebAuthnUser is exercised when store is non-nil.
	store := &mockStore{
		creds: []*Credential{
			{
				ID:           uuid.New(),
				CredentialID: []byte("cred-1"),
				PublicKey:    []byte("pubkey"),
				Counter:      1,
				Transports:   []string{"internal"},
			},
		},
	}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodPost, "/api/v1/webauthn/register/begin?user_id="+testUserIDStr, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.beginRegistration(rr, req)
	assertStatus(t, rr, http.StatusOK)
}

// ---------------------------------------------------------------------------
// finishRegistration
// ---------------------------------------------------------------------------

func TestFinishRegistration_Errors(t *testing.T) {
	h := testHandler(t, nil)

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodGet, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusMethodNotAllowed)
	})

	t.Run("missing tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, "")
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("missing user_id", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("malformed body", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, "not-json")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "parse credential creation")
	})

	t.Run("empty body", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "parse credential creation")
	})

	t.Run("session not found", func(t *testing.T) {
		body := buildRegBody("nonexistent-challenge-12345")
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, body)
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishRegistration(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "session expired or not found")
	})
}

func TestFinishRegistration_InvalidAttestation(t *testing.T) {
	h := testHandler(t, nil)

	// Step 1: begin registration to obtain a valid challenge + session.
	beginRR := httptest.NewRecorder()
	beginReq := newReq(http.MethodPost, "/api/v1/webauthn/register/begin?user_id="+testUserIDStr, "")
	beginReq.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.beginRegistration(beginRR, beginReq)
	assertStatus(t, beginRR, http.StatusOK)

	ch := extractChallenge(t, beginRR.Body.Bytes())

	// Step 2: finish registration with the matching challenge but fake attestation.
	// Parse will succeed, session will be found, but CreateCredential will fail.
	body := buildRegBody(ch)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodPost, "/api/v1/webauthn/register/finish?user_id="+testUserIDStr, body)
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.finishRegistration(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
	assertBodyContains(t, rr, "verify attestation")
}

// ---------------------------------------------------------------------------
// beginAuthentication
// ---------------------------------------------------------------------------

func TestBeginAuthentication_Errors(t *testing.T) {
	h := testHandler(t, nil)

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodGet, "/api/v1/webauthn/auth/begin", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.beginAuthentication(rr, req)
		assertStatus(t, rr, http.StatusMethodNotAllowed)
	})

	t.Run("missing tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/begin", "")
		h.beginAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("invalid tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/begin", "")
		req.Header.Set("X-Tenant-ID", "garbage")
		h.beginAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})
}

func TestBeginAuthentication_NoCredentials(t *testing.T) {
	// In go-webauthn v0.17, BeginLogin requires at least one credential.
	// The handler creates an ephemeral user with no credentials, so this returns 500.
	h := testHandler(t, nil)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodPost, "/api/v1/webauthn/auth/begin", "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.beginAuthentication(rr, req)
	assertStatus(t, rr, http.StatusInternalServerError)
	assertBodyContains(t, rr, "begin login")
}

// ---------------------------------------------------------------------------
// finishAuthentication
// ---------------------------------------------------------------------------

func TestFinishAuthentication_Errors(t *testing.T) {
	h := testHandler(t, nil)

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodGet, "/api/v1/webauthn/auth/finish", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishAuthentication(rr, req)
		assertStatus(t, rr, http.StatusMethodNotAllowed)
	})

	t.Run("missing tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/finish", "")
		h.finishAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("malformed body", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/finish", "garbage")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "parse assertion")
	})

	t.Run("empty body", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/finish", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "parse assertion")
	})

	t.Run("session not found", func(t *testing.T) {
		body := buildAuthBody("nonexistent-challenge-99999")
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/auth/finish", body)
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.finishAuthentication(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "session expired or not found")
	})
}

func TestFinishAuthentication_InvalidAssertion(t *testing.T) {
	h := testHandler(t, nil)

	// Manually inject a session since BeginLogin fails without credentials.
	tenantUUID := uuid.MustParse(testTenantIDStr)
	challenge := "manual-test-challenge"
	h.sessions.save("auth:"+challenge, &sessionData{
		tenantID:  tenantUUID,
		challenge: challenge,
		data: &wbn.SessionData{
			Challenge: challenge,
			UserID:    []byte(testUserIDStr),
		},
	})

	// finishAuthentication with matching challenge but fake assertion.
	// Parse succeeds, session found, but ValidateLogin fails.
	body := buildAuthBody(challenge)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodPost, "/api/v1/webauthn/auth/finish", body)
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.finishAuthentication(rr, req)
	assertStatus(t, rr, http.StatusUnauthorized)
	assertBodyContains(t, rr, "verify assertion")
}

// ---------------------------------------------------------------------------
// listCredentials
// ---------------------------------------------------------------------------

func TestListCredentials_Errors(t *testing.T) {
	h := testHandler(t, nil)

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/credentials", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.listCredentials(rr, req)
		assertStatus(t, rr, http.StatusMethodNotAllowed)
	})

	t.Run("missing tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodGet, "/api/v1/webauthn/credentials", "")
		h.listCredentials(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("invalid tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodGet, "/api/v1/webauthn/credentials", "")
		req.Header.Set("X-Tenant-ID", "xyz")
		h.listCredentials(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})
}

func TestListCredentials_NoUserID(t *testing.T) {
	h := testHandler(t, nil)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials", "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "credentials")
}

func TestListCredentials_NilStore(t *testing.T) {
	h := testHandler(t, nil)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials?user_id="+testUserIDStr, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "credentials")
}

func TestListCredentials_InvalidUserID(t *testing.T) {
	store := &mockStore{}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials?user_id=bad", "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	assertStatus(t, rr, http.StatusBadRequest)
	assertBodyContains(t, rr, "invalid user_id")
}

func TestListCredentials_StoreError(t *testing.T) {
	store := &mockStore{getByUserErr: errors.New("db down")}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials?user_id="+testUserIDStr, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	// Store error returns empty list, not an error status.
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "credentials")
}

func TestListCredentials_WithCreds(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		creds: []*Credential{
			{
				ID:           uuid.New(),
				Name:         "My Passkey",
				CredentialID: []byte("cred-bytes-1"),
				CreatedAt:    now,
			},
			{
				ID:           uuid.New(),
				Name:         "YubiKey",
				CredentialID: []byte("cred-bytes-2"),
				CreatedAt:    now,
			},
		},
	}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials?user_id="+testUserIDStr, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "My Passkey")
	assertBodyContains(t, rr, "YubiKey")
}

func TestListCredentials_EmptyCreds(t *testing.T) {
	store := &mockStore{creds: nil}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	req := newReq(http.MethodGet, "/api/v1/webauthn/credentials?user_id="+testUserID2Str, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.listCredentials(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "credentials")
}

// ---------------------------------------------------------------------------
// deleteCredential
// ---------------------------------------------------------------------------

func TestDeleteCredential_Errors(t *testing.T) {
	h := testHandler(t, nil)

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodPost, "/api/v1/webauthn/credentials/AAAA", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.deleteCredential(rr, req)
		assertStatus(t, rr, http.StatusMethodNotAllowed)
	})

	t.Run("missing tenant", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodDelete, "/api/v1/webauthn/credentials/AAAA", "")
		h.deleteCredential(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
	})

	t.Run("invalid credential ID", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := newReq(http.MethodDelete, "/api/v1/webauthn/credentials/!!!invalid-base64!!!", "")
		req.Header.Set("X-Tenant-ID", testTenantIDStr)
		h.deleteCredential(rr, req)
		assertStatus(t, rr, http.StatusBadRequest)
		assertBodyContains(t, rr, "invalid credential ID")
	})
}

func TestDeleteCredential_NilStore(t *testing.T) {
	// Skeleton mode (nil store) — should succeed without touching DB.
	h := testHandler(t, nil)
	rr := httptest.NewRecorder()
	credID := base64.RawURLEncoding.EncodeToString([]byte("test-cred-id"))
	req := newReq(http.MethodDelete, "/api/v1/webauthn/credentials/"+credID, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.deleteCredential(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "deleted")
}

func TestDeleteCredential_StoreError(t *testing.T) {
	store := &mockStore{deleteErr: errors.New("db locked")}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	credID := base64.RawURLEncoding.EncodeToString([]byte("test-cred-id"))
	req := newReq(http.MethodDelete, "/api/v1/webauthn/credentials/"+credID, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.deleteCredential(rr, req)
	assertStatus(t, rr, http.StatusInternalServerError)
	assertBodyContains(t, rr, "internal server error")
}

func TestDeleteCredential_Success(t *testing.T) {
	store := &mockStore{}
	h := testHandler(t, store)
	rr := httptest.NewRecorder()
	credID := base64.RawURLEncoding.EncodeToString([]byte("some-cred"))
	req := newReq(http.MethodDelete, "/api/v1/webauthn/credentials/"+credID, "")
	req.Header.Set("X-Tenant-ID", testTenantIDStr)
	h.deleteCredential(rr, req)
	assertStatus(t, rr, http.StatusOK)
	assertBodyContains(t, rr, "deleted")
}

// ---------------------------------------------------------------------------
// buildWebAuthnUser (exercised indirectly, but verify directly for coverage)
// ---------------------------------------------------------------------------

func TestBuildWebAuthnUser_NilStore(t *testing.T) {
	h := testHandler(t, nil)
	uid := uuid.MustParse(testUserIDStr)
	u, err := h.buildWebAuthnUser(context.Background(), uuid.MustParse(testTenantIDStr), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("user is nil")
	}
	if u.WebAuthnName() != uid.String() {
		t.Errorf("username = %q, want %q", u.WebAuthnName(), uid.String())
	}
	if len(u.WebAuthnCredentials()) != 0 {
		t.Errorf("expected 0 credentials, got %d", len(u.WebAuthnCredentials()))
	}
}

func TestBuildWebAuthnUser_WithStore(t *testing.T) {
	store := &mockStore{
		creds: []*Credential{
			{CredentialID: []byte("c1"), PublicKey: []byte("pk1"), Counter: 5},
		},
	}
	h := testHandler(t, store)
	uid := uuid.MustParse(testUserIDStr)
	u, err := h.buildWebAuthnUser(context.Background(), uuid.MustParse(testTenantIDStr), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	creds := u.WebAuthnCredentials()
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Authenticator.SignCount != 5 {
		t.Errorf("sign count = %d, want 5", creds[0].Authenticator.SignCount)
	}
}

func TestWebAuthnUser_DisplayName(t *testing.T) {
	uid := uuid.MustParse(testUserIDStr)
	u := &webAuthnUser{
		id:          uid,
		username:    "alice",
		displayName: "Alice Smith",
	}
	if u.WebAuthnDisplayName() != "Alice Smith" {
		t.Errorf("displayName = %q", u.WebAuthnDisplayName())
	}
	// WebAuthnID returns the 16-byte UUID.
	if len(u.WebAuthnID()) != 16 {
		t.Errorf("WebAuthnID length = %d, want 16", len(u.WebAuthnID()))
	}
}
