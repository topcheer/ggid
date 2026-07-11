package scim

// SCIM 2.0 PATCH Compliance Tests (RFC 7644)
// Verifies complex PATCH operations: enterprise extension, replace with filter,
// remove from nested array, multi-operation sequences.
// Date: 2026-07-25

import (
	"encoding/json"
	"testing"
)

// ========== RFC 7644 PATCH Compliance Tests ==========

// TestSCIMPatchCompliance_EnterpriseExtensionUpdate verifies replacing
// attributes on the EnterpriseUser extension schema. Since the extension URN
// contains dots (urn:...:2.0:User) which the path parser treats as nested
// separators, enterprise extension updates use the no-path replace approach:
// the value object is merged into the resource with the URN as a key.
func TestSCIMPatchCompliance_EnterpriseExtensionUpdate(t *testing.T) {
	entSchema := "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	attrs := map[string]any{
		"userName": "jdoe",
		"name": map[string]any{
			"givenName":  "John",
			"familyName": "Doe",
		},
		entSchema: map[string]any{
			"employeeNumber": "EMP-001",
			"department":     "Engineering",
			"division":       "Platform",
		},
	}

	// Replace the entire enterprise extension via no-path replace with the
	// extension URN as a key in the value object. The patch engine merges
	// the value's top-level keys into attrs.
	val, _ := json.Marshal(map[string]any{
		entSchema: map[string]any{
			"employeeNumber": "EMP-002",
			"department":     "Security",
		},
	})
	ops := []PatchOperation{
		{Op: "replace", Path: "", Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	entRaw, ok := result[entSchema]
	if !ok {
		t.Fatal("enterprise extension should exist after patch")
	}
	entMap, ok := entRaw.(map[string]any)
	if !ok {
		t.Fatalf("enterprise extension should be a map, got %T", entRaw)
	}

	if entMap["department"] != "Security" {
		t.Errorf("department should be 'Security', got %v", entMap["department"])
	}
	if entMap["employeeNumber"] != "EMP-002" {
		t.Errorf("employeeNumber should be 'EMP-002', got %v", entMap["employeeNumber"])
	}
}

// TestSCIMPatchCompliance_ReplaceWithComplexFilter verifies replacing a specific
// sub-attribute of a multi-valued attribute using a filter expression.
func TestSCIMPatchCompliance_ReplaceWithComplexFilter(t *testing.T) {
	attrs := map[string]any{
		"userName": "asmith",
		"emails": []any{
			map[string]any{"value": "asmith@work.com", "type": "work", "primary": true},
			map[string]any{"value": "asmith@home.com", "type": "home"},
			map[string]any{"value": "asmith@old.com", "type": "work"},
		},
	}

	// Replace the value of the work email that matches old.com
	val, _ := json.Marshal("asmith@newwork.com")
	ops := []PatchOperation{
		{Op: "replace", Path: `emails[value eq "asmith@old.com"].value`, Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	emails, ok := result["emails"].([]any)
	if !ok {
		t.Fatal("emails should be an array")
	}

	foundUpdated := false
	for _, e := range emails {
		em := e.(map[string]any)
		if em["value"] == "asmith@newwork.com" {
			foundUpdated = true
			if em["type"] != "work" {
				t.Errorf("type should remain 'work', got %v", em["type"])
			}
		}
	}
	if !foundUpdated {
		t.Error("email value should have been updated to asmith@newwork.com")
	}

	// Other emails should be unchanged
	foundOriginal := false
	for _, e := range emails {
		em := e.(map[string]any)
		if em["value"] == "asmith@work.com" {
			foundOriginal = true
		}
	}
	if !foundOriginal {
		t.Error("original work email should still exist")
	}
}

// TestSCIMPatchCompliance_RemoveFromNestedArray verifies removing elements
// from a multi-valued attribute using a filter.
func TestSCIMPatchCompliance_RemoveFromNestedArray(t *testing.T) {
	attrs := map[string]any{
		"userName": "bjones",
		"emails": []any{
			map[string]any{"value": "bjones@work.com", "type": "work"},
			map[string]any{"value": "bjones@home.com", "type": "home"},
			map[string]any{"value": "bjones@temp.com", "type": "temp"},
			map[string]any{"value": "bjones@old.com", "type": "temp"},
		},
	}

	// Remove all temp-type emails
	ops := []PatchOperation{
		{Op: "remove", Path: `emails[type eq "temp"]`},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	emails, ok := result["emails"].([]any)
	if !ok {
		t.Fatal("emails should still be an array after remove")
	}

	if len(emails) != 2 {
		t.Fatalf("expected 2 emails after removing temp type, got %d", len(emails))
	}

	for _, e := range emails {
		em := e.(map[string]any)
		if em["type"] == "temp" {
			t.Error("no temp emails should remain after remove")
		}
	}
}

// TestSCIMPatchCompliance_MultiOperationSequence verifies a sequence of
// add, replace, and remove operations in a single PATCH request.
func TestSCIMPatchCompliance_MultiOperationSequence(t *testing.T) {
	attrs := map[string]any{
		"userName":   "mwilson",
		"displayName": "Mike Wilson",
		"active":      true,
		"emails": []any{
			map[string]any{"value": "mwilson@old.com", "type": "work"},
		},
		"name": map[string]any{
			"givenName":  "Mike",
			"familyName": "Wilson",
		},
	}

	phoneVal, _ := json.Marshal([]any{
		map[string]any{"value": "+1-555-0100", "type": "mobile"},
	})
	newEmailVal, _ := json.Marshal(map[string]any{
		"value": "mwilson@new.com", "type": "work",
	})
	displayVal, _ := json.Marshal("Michael Wilson")

	ops := []PatchOperation{
		// Add phone numbers
		{Op: "add", Path: "phoneNumbers", Value: phoneVal},
		// Replace display name
		{Op: "replace", Path: "displayName", Value: displayVal},
		// Remove old email
		{Op: "remove", Path: `emails[value eq "mwilson@old.com"]`},
		// Add new email
		{Op: "add", Path: "emails", Value: newEmailVal},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	// Phone numbers should exist
	phones, ok := result["phoneNumbers"].([]any)
	if !ok || len(phones) != 1 {
		t.Errorf("expected 1 phone number, got %v", result["phoneNumbers"])
	}

	// Display name should be updated
	if result["displayName"] != "Michael Wilson" {
		t.Errorf("displayName should be 'Michael Wilson', got %v", result["displayName"])
	}

	// Old email should be removed, new email added
	emails, ok := result["emails"].([]any)
	if !ok {
		t.Fatal("emails should exist")
	}
	for _, e := range emails {
		em := e.(map[string]any)
		if em["value"] == "mwilson@old.com" {
			t.Error("old email should have been removed")
		}
	}

	foundNew := false
	for _, e := range emails {
		em := e.(map[string]any)
		if em["value"] == "mwilson@new.com" {
			foundNew = true
		}
	}
	if !foundNew {
		t.Error("new email should have been added")
	}
}

// TestSCIMPatchCompliance_NestedPathReplace verifies replacing a nested
// sub-attribute (e.g., name.familyName).
func TestSCIMPatchCompliance_NestedPathReplace(t *testing.T) {
	attrs := map[string]any{
		"userName": "testuser",
		"name": map[string]any{
			"givenName":  "Test",
			"familyName": "User",
		},
	}

	val, _ := json.Marshal("Smith")
	ops := []PatchOperation{
		{Op: "replace", Path: "name.familyName", Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	nameMap, ok := result["name"].(map[string]any)
	if !ok {
		t.Fatal("name should be a map")
	}
	if nameMap["familyName"] != "Smith" {
		t.Errorf("familyName should be 'Smith', got %v", nameMap["familyName"])
	}
	if nameMap["givenName"] != "Test" {
		t.Errorf("givenName should remain 'Test', got %v", nameMap["givenName"])
	}
}

// TestSCIMPatchCompliance_URNColonNotation verifies that RFC 7644 colon notation
// for URN paths works (e.g., urn:...:User:department instead of urn:...:User.department).
func TestSCIMPatchCompliance_URNColonNotation(t *testing.T) {
	entSchema := "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	attrs := map[string]any{
		"userName": "colonuser",
		entSchema: map[string]any{
			"department":     "Engineering",
			"employeeNumber": "EMP-100",
		},
	}

	// Use colon notation: urn:...:User:department
	val, _ := json.Marshal("Security")
	ops := []PatchOperation{
		{Op: "replace", Path: entSchema + ":department", Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch with colon notation failed: %v", err)
	}

	entRaw, ok := result[entSchema]
	if !ok {
		t.Fatal("enterprise extension should exist")
	}
	entMap := entRaw.(map[string]any)
	if entMap["department"] != "Security" {
		t.Errorf("department should be 'Security' with colon notation, got %v", entMap["department"])
	}
	if entMap["employeeNumber"] != "EMP-100" {
		t.Errorf("employeeNumber should remain 'EMP-100', got %v", entMap["employeeNumber"])
	}
}

// TestSCIMPatchCompliance_URNColonNotationNested verifies colon notation with
// deeper nesting: urn:...:User:manager.displayName.
func TestSCIMPatchCompliance_URNColonNotationNested(t *testing.T) {
	entSchema := "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	attrs := map[string]any{
		"userName": "nesteduser",
		entSchema: map[string]any{
			"manager": map[string]any{
				"value":      "mgr-001",
				"displayName": "Old Manager",
			},
		},
	}

	// Colon notation with dotted sub-path: urn:...:User:manager.displayName
	val, _ := json.Marshal("New Manager")
	ops := []PatchOperation{
		{Op: "replace", Path: entSchema + ":manager.displayName", Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	entRaw := result[entSchema].(map[string]any)
	mgr := entRaw["manager"].(map[string]any)
	if mgr["displayName"] != "New Manager" {
		t.Errorf("manager displayName should be 'New Manager', got %v", mgr["displayName"])
	}
}

// TestSCIMPatchCompliance_UnsupportedOperation verifies that an unsupported
// PATCH operation returns an error.
func TestSCIMPatchCompliance_UnsupportedOperation(t *testing.T) {
	attrs := map[string]any{"userName": "test"}
	ops := []PatchOperation{
		{Op: "invalidOp", Path: "userName"},
	}

	_, err := ApplyPatch(attrs, ops)
	if err == nil {
		t.Error("ApplyPatch should error on unsupported operation")
	}
}

// TestSCIMPatchCompliance_EnterpriseManagerUpdate verifies updating the
// manager sub-object in the EnterpriseUser extension via no-path replace.
func TestSCIMPatchCompliance_EnterpriseManagerUpdate(t *testing.T) {
	entSchema := "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	attrs := map[string]any{
		"userName": "testuser",
		entSchema: map[string]any{
			"department": "Engineering",
			"manager": map[string]any{
				"value":      "old-manager-id",
				"displayName": "Old Boss",
			},
		},
	}

	// Update the enterprise extension via no-path replace with the URN as key.
	val, _ := json.Marshal(map[string]any{
		entSchema: map[string]any{
			"department": "Engineering",
			"manager": map[string]any{
				"value":      "new-manager-id",
				"displayName": "New Boss",
			},
		},
	})
	ops := []PatchOperation{
		{Op: "replace", Path: "", Value: val},
	}

	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	entRaw, ok := result[entSchema]
	if !ok {
		t.Fatal("enterprise extension should exist")
	}
	entMap := entRaw.(map[string]any)
	mgr, ok := entMap["manager"].(map[string]any)
	if !ok {
		t.Fatal("manager should be a map")
	}
	if mgr["value"] != "new-manager-id" {
		t.Errorf("manager value should be 'new-manager-id', got %v", mgr["value"])
	}
	if mgr["displayName"] != "New Boss" {
		t.Errorf("manager displayName should be 'New Boss', got %v", mgr["displayName"])
	}
	if entMap["department"] != "Engineering" {
		t.Errorf("department should remain 'Engineering', got %v", entMap["department"])
	}
}
