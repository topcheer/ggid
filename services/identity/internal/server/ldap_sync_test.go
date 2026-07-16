package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// resetLDAPState clears all LDAP stores between tests.
func resetLDAPState() {
	ldapConfigStore.Lock()
	ldapConfigStore.config = nil
	ldapConfigStore.Unlock()

	ldapSyncState.Lock()
	ldapSyncState.status = "never"
	ldapSyncState.lastRun = time.Time{}
	ldapSyncState.synced = 0
	ldapSyncState.totalFound = 0
	ldapSyncState.errs = []map[string]any{}
	ldapSyncState.Unlock()
}

func TestLDAPSyncConfig_GetUnconfigured(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/ldap/sync-config", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	cfg, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatal("missing config in response")
	}
	if cfg["user_filter"] != "(objectClass=person)" {
		t.Errorf("expected default user_filter, got %v", cfg["user_filter"])
	}
}

func TestLDAPSyncConfig_PutValid(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	body := `{"server_url":"ldap://test:389","bind_dn":"cn=admin","bind_password":"secret","base_dn":"dc=example,dc=com","user_filter":"(objectClass=person)","group_filter":"(objectClass=group)","start_tls":true,"auto_provision":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/identity/ldap/sync-config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "saved" {
		t.Errorf("expected status saved, got %v", resp["status"])
	}
}

func TestLDAPSyncConfig_PutMissingServerURL(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	body := `{"bind_dn":"cn=admin","base_dn":"dc=example,dc=com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/identity/ldap/sync-config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestLDAPSyncConfig_PutMissingBaseDN(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	body := `{"server_url":"ldap://test:389","bind_dn":"cn=admin"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/identity/ldap/sync-config", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestLDAPSyncConfig_MethodNotAllowed(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/identity/ldap/sync-config", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfig(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestLDAPSyncConfigTest_NoConfig(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/ldap/sync-config/test", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncConfigTest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no config, got %d", rr.Code)
	}
}

func TestLDAPSyncStatus_Unconfigured(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/ldap/sync-status", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["configured"] != false {
		t.Errorf("expected configured=false, got %v", resp["configured"])
	}
	if resp["status"] != "never" {
		t.Errorf("expected status=never, got %v", resp["status"])
	}
}

func TestLDAPSyncHistory_Empty(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/ldap/sync-history", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestLDAPSyncHistory_MethodNotAllowed(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/ldap/sync-history", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSyncHistory(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestLDAPSync_NoConfig(t *testing.T) {
	resetLDAPState()
	h := &HTTPHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/ldap/sync", nil)
	rr := httptest.NewRecorder()
	h.handleLDAPSync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no config, got %d", rr.Code)
	}
}
