package scim

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- SCIM-20: userName uniqueness + validation tests ---

func TestCreateUser_EmptyUserName(t *testing.T) {
	h := newTestHandler()
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":""}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty userName, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["scimType"] != ScimTypeInvalidSyntax {
		t.Errorf("expected scimType %s, got %v", ScimTypeInvalidSyntax, resp["scimType"])
	}
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	h := newTestHandler()
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"testuser","emails":[{"value":"not-an-email"}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["scimType"] != ScimTypeInvalidValue {
		t.Errorf("expected scimType %s, got %v", ScimTypeInvalidValue, resp["scimType"])
	}
}

func TestCreateUser_MissingUserName(t *testing.T) {
	h := newTestHandler()
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"emails":[{"value":"test@test.com"}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_ValidEmail(t *testing.T) {
	h := newTestHandler()
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"validuser","emails":[{"value":"valid@test.com"}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	// Should not be a 400 invalidValue — may be 409/500 from nil service but NOT invalidValue
	if w.Code == http.StatusBadRequest {
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["scimType"] == ScimTypeInvalidValue {
			t.Error("valid email should not trigger invalidValue error")
		}
	}
}

func TestCreateUser_NoEmail(t *testing.T) {
	h := newTestHandler()
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"noemail"}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	// Should not be a 400 invalidValue
	if w.Code == http.StatusBadRequest {
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["scimType"] == ScimTypeInvalidValue {
			t.Error("missing email should not trigger invalidValue error")
		}
	}
}

func TestCreateUser_MalformedJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader("{invalid json}"))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleUsersCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

// --- SCIM-16: PATCH group operations tests ---

func TestPatchGroup_AddMembers(t *testing.T) {
	h := newTestHandler()
	body := `{"Operations":[{"op":"add","path":"members","value":[{"value":"user-1","display":"Alice"}]}]}`
	req := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.patchGroup(w, req, "role-admin-001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var group SCIMGroup
	json.Unmarshal(w.Body.Bytes(), &group)
	if len(group.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(group.Members))
	}
	if group.Members[0].Value != "user-1" {
		t.Errorf("expected user-1, got %s", group.Members[0].Value)
	}
}

func TestPatchGroup_ReplaceDisplayName(t *testing.T) {
	h := newTestHandler()
	body := `{"Operations":[{"op":"replace","path":"displayName","value":"Super Admin"}]}`
	req := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.patchGroup(w, req, "role-admin-001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var group SCIMGroup
	json.Unmarshal(w.Body.Bytes(), &group)
	if group.DisplayName != "Super Admin" {
		t.Errorf("expected 'Super Admin', got '%s'", group.DisplayName)
	}
}

func TestPatchGroup_ReplaceAllMembers(t *testing.T) {
	h := newTestHandler()
	body := `{"Operations":[{"op":"replace","path":"members","value":[{"value":"u1"},{"value":"u2"}]}]}`
	req := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.patchGroup(w, req, "role-admin-001")

	var group SCIMGroup
	json.Unmarshal(w.Body.Bytes(), &group)
	if len(group.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(group.Members))
	}
}

func TestPatchGroup_AddDuplicateMembers(t *testing.T) {
	h := newTestHandler()
	// First add a member
	body1 := `{"Operations":[{"op":"add","path":"members","value":[{"value":"dup-1"}]}]}`
	req1 := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(body1))
	w1 := httptest.NewRecorder()
	h.patchGroup(w1, req1, "role-admin-001")

	// Add the same member again
	body2 := `{"Operations":[{"op":"add","path":"members","value":[{"value":"dup-1"}]}]}`
	req2 := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(body2))
	w2 := httptest.NewRecorder()
	h.patchGroup(w2, req2, "role-admin-001")

	var group SCIMGroup
	json.Unmarshal(w2.Body.Bytes(), &group)
	if len(group.Members) != 1 {
		t.Errorf("expected 1 member (dedup), got %d", len(group.Members))
	}
}

func TestPatchGroup_RemoveMemberByFilter(t *testing.T) {
	h := newTestHandler()
	// Add members first
	addBody := `{"Operations":[{"op":"add","path":"members","value":[{"value":"rm-1"},{"value":"rm-2"}]}]}`
	addReq := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(addBody))
	addW := httptest.NewRecorder()
	h.patchGroup(addW, addReq, "role-admin-001")

	// Remove one member by filter
	rmBody := `{"Operations":[{"op":"remove","path":"members[value eq \"rm-1\"]"}]}`
	rmReq := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(rmBody))
	rmW := httptest.NewRecorder()
	h.patchGroup(rmW, rmReq, "role-admin-001")

	var group SCIMGroup
	json.Unmarshal(rmW.Body.Bytes(), &group)
	if len(group.Members) != 1 {
		t.Errorf("expected 1 member after removal, got %d", len(group.Members))
	}
	if group.Members[0].Value != "rm-2" {
		t.Errorf("expected rm-2 to remain, got %s", group.Members[0].Value)
	}
}

func TestPatchGroup_RemoveAllMembers(t *testing.T) {
	h := newTestHandler()
	// Add members first
	addBody := `{"Operations":[{"op":"add","path":"members","value":[{"value":"x1"},{"value":"x2"}]}]}`
	addReq := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(addBody))
	addW := httptest.NewRecorder()
	h.patchGroup(addW, addReq, "role-admin-001")

	// Remove all
	rmBody := `{"Operations":[{"op":"remove","path":"members"}]}`
	rmReq := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader(rmBody))
	rmW := httptest.NewRecorder()
	h.patchGroup(rmW, rmReq, "role-admin-001")

	var group SCIMGroup
	json.Unmarshal(rmW.Body.Bytes(), &group)
	if len(group.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(group.Members))
	}
}

func TestPatchGroup_GroupNotFound(t *testing.T) {
	h := newTestHandler()
	body := `{"Operations":[{"op":"replace","path":"displayName","value":"Test"}]}`
	req := httptest.NewRequest("PATCH", "/scim/v2/Groups/nonexistent", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.patchGroup(w, req, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPatchGroup_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("PATCH", "/scim/v2/Groups/role-admin-001", strings.NewReader("{bad}"))
	w := httptest.NewRecorder()
	h.patchGroup(w, req, "role-admin-001")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Helper for test handler ---

const testTenantID = "00000000-0000-0000-0000-000000000001"

func newTestHandler() *Handler {
	// Reset mutable group store for test isolation
	patchGroupStore = map[string]*SCIMGroup{}
	return &Handler{}
}

func TestValueToMembers(t *testing.T) {
	val := []any{
		map[string]any{"value": "u1", "display": "Alice", "type": "User"},
		map[string]any{"value": "u2", "display": "Bob"},
	}
	members := valueToMembers(val)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Value != "u1" {
		t.Error("first member value should be u1")
	}
	if members[0].Display != "Alice" {
		t.Error("first member display should be Alice")
	}
}

func TestValueToMembers_InvalidInput(t *testing.T) {
	members := valueToMembers("not-an-array")
	if members != nil {
		t.Error("expected nil for non-array input")
	}
}

func TestParseMemberFilter_SingleValue(t *testing.T) {
	result := parseMemberFilter(`members[value eq "abc"]`)
	if !result["abc"] {
		t.Error("should contain abc")
	}
}

func TestParseMemberFilter_MultipleValues(t *testing.T) {
	result := parseMemberFilter(`members[value eq "abc" or value eq "def"]`)
	if !result["abc"] || !result["def"] {
		t.Error("should contain both abc and def")
	}
}

func TestParseMemberFilter_NoBrackets(t *testing.T) {
	result := parseMemberFilter("members")
	if len(result) != 0 {
		t.Error("should return empty map when no brackets")
	}
}
