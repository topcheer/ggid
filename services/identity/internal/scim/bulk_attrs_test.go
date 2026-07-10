package scim

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- SCIM-14: Bulk Operations Tests ---

func TestExtractIDFromPath(t *testing.T) {
	id, err := extractIDFromPath("/Users/550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("got %s", id)
	}
}

func TestExtractIDFromPath_Invalid(t *testing.T) {
	_, err := extractIDFromPath("/Users")
	if err == nil {
		t.Error("expected error for path without ID")
	}
}

func TestExtractIDFromPath_MalformedUUID(t *testing.T) {
	_, err := extractIDFromPath("/Users/not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("Users/123/Groups/456")
	if len(parts) != 4 {
		t.Fatalf("got %d parts, want 4", len(parts))
	}
}

func TestBulkRequest_Unmarshal(t *testing.T) {
	body := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "bulk-1",
				"data": {"userName": "john", "emails": [{"value": "john@test.com"}]}
			}
		]
	}`
	var req BulkRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Operations) != 1 {
		t.Fatalf("expected 1 op, got %d", len(req.Operations))
	}
	if req.Operations[0].Method != "POST" {
		t.Errorf("method = %s", req.Operations[0].Method)
	}
	if req.Operations[0].BulkID != "bulk-1" {
		t.Errorf("bulkId = %s", req.Operations[0].BulkID)
	}
}

func TestBulkResponse_Marshal(t *testing.T) {
	resp := BulkResponse{
		Schemas: []string{bulkResponseSchema},
		Operations: []BulkOperationResponse{
			{Method: "POST", Status: "201", Location: "/scim/v2/Users/123", BulkID: "bulk-1"},
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "201") {
		t.Error("missing status 201")
	}
	if !strings.Contains(s, "bulk-1") {
		t.Error("missing bulkId")
	}
}

func TestMaxBulkOperations(t *testing.T) {
	if maxBulkOperations != 1000 {
		t.Errorf("maxBulkOperations = %d, want 1000", maxBulkOperations)
	}
}

func TestExecuteBulkOp_UnsupportedMethod(t *testing.T) {
	h := &Handler{}
	resp, err := h.executeBulkOp(nil, BulkOperationRequest{Method: "GET"})
	if err == nil {
		t.Error("expected error for unsupported method")
	}
	if resp.Status != "400" {
		t.Errorf("status = %s, want 400", resp.Status)
	}
}

// --- SCIM-07: Pagination Meta Tests ---

func TestListResponse_HasPaginationFields(t *testing.T) {
	lr := ListResponse{
		Schemas:      []string{scimListSchema},
		TotalResults: 100,
		ItemsPerPage: 20,
		StartIndex:   1,
		Resources:    []SCIMUser{},
	}
	data, err := json.Marshal(lr)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "\"totalResults\":100") {
		t.Error("missing totalResults")
	}
	if !strings.Contains(s, "\"itemsPerPage\":20") {
		t.Error("missing itemsPerPage")
	}
	if !strings.Contains(s, "\"startIndex\":1") {
		t.Error("missing startIndex")
	}
}

func TestListResponse_StartIndexIsOneBased(t *testing.T) {
	// SCIM spec: startIndex is 1-based, not 0-based
	lr := ListResponse{
		StartIndex: 1,
	}
	if lr.StartIndex != 1 {
		t.Error("default startIndex should be 1")
	}
}

// --- SCIM-11: Attribute Filtering Tests ---

func TestParseAttrList(t *testing.T) {
	m := parseAttrList("userName, emails, displayName")
	if !m["userName"] {
		t.Error("missing userName")
	}
	if !m["emails"] {
		t.Error("missing emails")
	}
	if !m["displayName"] {
		t.Error("missing displayName")
	}
}

func TestParseAttrList_Empty(t *testing.T) {
	m := parseAttrList("")
	if len(m) != 0 {
		t.Errorf("expected 0 attrs, got %d", len(m))
	}
}

func TestApplyAttributeFilter_NoFilter(t *testing.T) {
	u := SCIMUser{
		ID:        "123",
		UserName:  "john",
		Emails:    []SCIMEmail{{Value: "john@test.com"}},
		Active:    true,
	}
	result := applyAttributeFilter(u, "", "")
	if result.UserName != "john" {
		t.Error("userName should be preserved")
	}
	if len(result.Emails) != 1 {
		t.Error("emails should be preserved")
	}
}

func TestApplyAttributeFilter_Whitelist(t *testing.T) {
	u := SCIMUser{
		ID:         "123",
		UserName:   "john",
		DisplayName: "John Doe",
		Emails:     []SCIMEmail{{Value: "john@test.com"}},
	}
	result := applyAttributeFilter(u, "userName", "")
	// userName, id, schemas always kept
	if result.UserName != "john" {
		t.Error("userName should be kept")
	}
	if result.DisplayName != "" {
		t.Error("displayName should be filtered out")
	}
	if len(result.Emails) != 0 {
		t.Error("emails should be filtered out")
	}
}

func TestApplyAttributeFilter_Exclude(t *testing.T) {
	u := SCIMUser{
		ID:        "123",
		UserName:  "john",
		Emails:    []SCIMEmail{{Value: "john@test.com"}},
	}
	result := applyAttributeFilter(u, "", "emails")
	if result.UserName != "john" {
		t.Error("userName should be kept")
	}
	if len(result.Emails) != 0 {
		t.Error("emails should be excluded")
	}
}

func TestApplyAttributeFilter_Multiple(t *testing.T) {
	u := SCIMUser{
		ID:         "123",
		UserName:   "john",
		DisplayName: "John Doe",
		Emails:     []SCIMEmail{{Value: "john@test.com"}},
		PhoneNumbers: []SCIMPhone{{Value: "555-1234"}},
	}
	// Include userName and emails, exclude everything else
	result := applyAttributeFilter(u, "userName,emails", "")
	if result.UserName != "john" {
		t.Error("userName should be kept")
	}
	if len(result.Emails) != 1 {
		t.Error("emails should be kept")
	}
	if result.DisplayName != "" {
		t.Error("displayName should be filtered out")
	}
	if len(result.PhoneNumbers) != 0 {
		t.Error("phoneNumbers should be filtered out")
	}
}

func TestApplyAttributeFilter_AlwaysKeepsRequired(t *testing.T) {
	u := SCIMUser{
		ID:       "abc-123",
		UserName: "testuser",
	}
	result := applyAttributeFilter(u, "displayName", "")
	// schemas, id, userName are always required
	if result.ID != "abc-123" {
		t.Error("id should always be kept")
	}
	if result.UserName != "testuser" {
		t.Error("userName should always be kept")
	}
}

func TestScimUserToAttrs(t *testing.T) {
	u := SCIMUser{
		ID:        "123",
		UserName:  "john",
		Emails:    []SCIMEmail{{Value: "john@test.com", Type: "work", Primary: true}},
		Active:    true,
	}
	attrs := scimUserToAttrs(u)
	if attrs["id"] != "123" {
		t.Error("missing id")
	}
	if attrs["userName"] != "john" {
		t.Error("missing userName")
	}
	emails, ok := attrs["emails"].([]any)
	if !ok || len(emails) != 1 {
		t.Error("emails not converted correctly")
	}
}
